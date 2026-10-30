package integrations

import (
	"context"
	"log"
	"strconv"
	"time"

	"multicatalogo-backend/config"
)

func RunNotificationWorker(ctx context.Context, sender NotificationSender) {
	if sender == nil || config.DB == nil {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := deliverDueNotifications(ctx, sender); err != nil && ctx.Err() == nil {
			log.Printf("notification queue cycle failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func deliverDueNotifications(ctx context.Context, sender NotificationSender) error {
	tx, err := config.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `UPDATE dental_notification_outbox SET status='retry',locked_at=NULL,scheduled_at=NOW() WHERE status='sending' AND locked_at < NOW()-INTERVAL '10 minutes'`)
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,channel,recipient,template_key,message,scheduled_at FROM dental_notification_outbox WHERE status IN ('pending','retry') AND scheduled_at<=NOW() ORDER BY scheduled_at FOR UPDATE SKIP LOCKED LIMIT 50`)
	if err != nil {
		return err
	}
	items := make([]Delivery, 0)
	for rows.Next() {
		var item Delivery
		if err := rows.Scan(&item.ID, &item.Channel, &item.Recipient, &item.Template, &item.Message, &item.Scheduled); err != nil {
			rows.Close()
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, item := range items {
		if _, err := tx.ExecContext(ctx, `UPDATE dental_notification_outbox SET status='sending',attempts=attempts+1,locked_at=NOW() WHERE id=$1`, item.ID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, item := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		reference, sendErr := sender.Send(ctx, item)
		if sendErr != nil {
			_, err = config.DB.ExecContext(ctx, `UPDATE dental_notification_outbox SET status=CASE WHEN attempts>=5 THEN 'needs_configuration' ELSE 'retry' END,scheduled_at=NOW()+LEAST(60,POWER(2,attempts))*INTERVAL '1 minute',locked_at=NULL,last_error='provider delivery failed' WHERE id=$1`, item.ID)
			if err != nil {
				return err
			}
			log.Printf("notification %s delivery failed", strconv.FormatInt(item.ID, 10))
			continue
		}
		_, err = config.DB.ExecContext(ctx, `UPDATE dental_notification_outbox SET status='sent',sent_at=NOW(),locked_at=NULL,last_error=NULL,provider_reference=NULLIF($2,'') WHERE id=$1`, item.ID, reference)
		if err != nil {
			return err
		}
	}
	return nil
}
