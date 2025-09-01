package internal

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct{ Pool *pgxpool.Pool }

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{Pool: pool} }

// Upsert — одна транзакция:
// 1) upsert шапки заказа
// 2) upsert delivery
// 3) upsert payment
// 4) replace items (delete + batch insert)
func (r *Repo) Upsert(ctx context.Context, o *Order) error {
	tx, err := r.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		// rollback по умолчанию, commit явно ниже
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// orders
	_, err = tx.Exec(ctx, `
		INSERT INTO orders(
		  order_uid, track_number, entry, locale, internal_signature,
		  customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11, now())
		ON CONFLICT(order_uid) DO UPDATE SET
		  track_number=EXCLUDED.track_number,
		  entry=EXCLUDED.entry,
		  locale=EXCLUDED.locale,
		  internal_signature=EXCLUDED.internal_signature,
		  customer_id=EXCLUDED.customer_id,
		  delivery_service=EXCLUDED.delivery_service,
		  shardkey=EXCLUDED.shardkey,
		  sm_id=EXCLUDED.sm_id,
		  date_created=EXCLUDED.date_created,
		  oof_shard=EXCLUDED.oof_shard,
		  updated_at=now()
	`, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature,
		o.CustomerID, o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		return err
	}

	// deliveries (1:1)
	_, err = tx.Exec(ctx, `
		INSERT INTO deliveries(
		  order_uid, name, phone, zip, city, address, region, email
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(order_uid) DO UPDATE SET
		  name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip,
		  city=EXCLUDED.city, address=EXCLUDED.address, region=EXCLUDED.region, email=EXCLUDED.email
	`, o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return err
	}

	// payments (1:1)
	_, err = tx.Exec(ctx, `
		INSERT INTO payments(
		  order_uid, transaction, request_id, currency, provider,
		  amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT(order_uid) DO UPDATE SET
		  transaction=EXCLUDED.transaction,
		  request_id=EXCLUDED.request_id,
		  currency=EXCLUDED.currency,
		  provider=EXCLUDED.provider,
		  amount=EXCLUDED.amount,
		  payment_dt=EXCLUDED.payment_dt,
		  bank=EXCLUDED.bank,
		  delivery_cost=EXCLUDED.delivery_cost,
		  goods_total=EXCLUDED.goods_total,
		  custom_fee=EXCLUDED.custom_fee
	`, o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return err
	}

	// items — удаление и вставка
	if _, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID); err != nil {
		return err
	}

	// INSERTЫ
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO items(
			  order_uid, chrt_id, track_number, price, rid, name,
			  sale, size, total_price, nm_id, brand, status
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT(order_uid, chrt_id) DO UPDATE SET
			  track_number=EXCLUDED.track_number,
			  price=EXCLUDED.price,
			  rid=EXCLUDED.rid,
			  name=EXCLUDED.name,
			  sale=EXCLUDED.sale,
			  size=EXCLUDED.size,
			  total_price=EXCLUDED.total_price,
			  nm_id=EXCLUDED.nm_id,
			  brand=EXCLUDED.brand,
			  status=EXCLUDED.status
		`, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name,
			it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	tx = nil
	return nil
}

func (r *Repo) Get(ctx context.Context, id string) (*Order, bool, error) {
	// orders
	var o Order
	err := r.Pool.QueryRow(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature,
		       customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid=$1
	`, id).Scan(
		&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
		&o.CustomerID, &o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}

	// deliveries
	err = r.Pool.QueryRow(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid=$1
	`, id).Scan(
		&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	// payments
	err = r.Pool.QueryRow(ctx, `
		SELECT transaction, request_id, currency, provider,
		       amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid=$1
	`, id).Scan(
		&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost,
		&o.Payment.GoodsTotal, &o.Payment.CustomFee,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	// items
	rows, err := r.Pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid=$1 ORDER BY chrt_id
	`, id)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(
			&it.ChrtID, &it.TrackNumber, &it.Price, &it.RID, &it.Name,
			&it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status,
		); err != nil {
			return nil, false, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	o.Items = items

	return &o, true, nil
}

// тут прогремаю кэш - беру последние N заказов по updated_at
func (r *Repo) LoadRecent(ctx context.Context, n int) ([]*Order, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT order_uid
		FROM orders
		ORDER BY updated_at DESC
		LIMIT $1
	`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]*Order, 0, len(ids))
	for _, id := range ids {
		o, ok, err := r.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *Repo) ListRecentIDs(ctx context.Context, n int) ([]string, error) {
	rows, err := r.Pool.Query(ctx, `SELECT order_uid FROM orders ORDER BY updated_at DESC LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
