package nutrition

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/postgres"
)

// BarcodeProduct is a cached food product keyed by its barcode.
type BarcodeProduct struct {
	Barcode      string
	Name         string
	Brand        string
	ImageURL     string
	Per100       Macros
	ServingGrams *float64
	Source       string
}

type BarcodeRepo struct{ db *postgres.DB }

func NewBarcodeRepo(db *postgres.DB) *BarcodeRepo { return &BarcodeRepo{db: db} }

const barcodeCols = `barcode, name, brand, image_url,
	kcal_per_100g::float8, protein_per_100g::float8, fat_per_100g::float8, carbs_per_100g::float8,
	serving_grams::float8, source`

func scanBarcode(row pgx.Row) (*BarcodeProduct, error) {
	var p BarcodeProduct
	if err := row.Scan(&p.Barcode, &p.Name, &p.Brand, &p.ImageURL,
		&p.Per100.Kcal, &p.Per100.Protein, &p.Per100.Fat, &p.Per100.Carbs,
		&p.ServingGrams, &p.Source); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *BarcodeRepo) Get(ctx context.Context, barcode string) (*BarcodeProduct, error) {
	return scanBarcode(r.db.Read().QueryRow(ctx, `SELECT `+barcodeCols+` FROM food_barcodes WHERE barcode=$1`, barcode))
}

// Put upserts a product into the shared cache.
func (r *BarcodeRepo) Put(ctx context.Context, p BarcodeProduct) error {
	const q = `
		INSERT INTO food_barcodes (barcode, name, brand, image_url, kcal_per_100g, protein_per_100g,
			fat_per_100g, carbs_per_100g, serving_grams, source, fetched_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())
		ON CONFLICT (barcode) DO UPDATE SET
			name=EXCLUDED.name, brand=EXCLUDED.brand, image_url=EXCLUDED.image_url,
			kcal_per_100g=EXCLUDED.kcal_per_100g, protein_per_100g=EXCLUDED.protein_per_100g,
			fat_per_100g=EXCLUDED.fat_per_100g, carbs_per_100g=EXCLUDED.carbs_per_100g,
			serving_grams=EXCLUDED.serving_grams, source=EXCLUDED.source, fetched_at=now()`
	_, err := r.db.Pool.Exec(ctx, q, p.Barcode, p.Name, p.Brand, p.ImageURL,
		p.Per100.Kcal, p.Per100.Protein, p.Per100.Fat, p.Per100.Carbs, p.ServingGrams, p.Source)
	return err
}
