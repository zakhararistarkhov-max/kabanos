// Package openfoodfacts is a tiny read-only client for the Open Food Facts API
// (https://openfoodfacts.org), used to resolve a product barcode into a name and
// per-100g macros. It is deliberately minimal: one lookup call, normalized into
// our own shape, so the rest of the app never depends on OFF's JSON.
package openfoodfacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound means OFF has no product for the barcode.
var ErrNotFound = errors.New("product not found")

// Product is the normalized result of a lookup.
type Product struct {
	Barcode       string
	Name          string
	Brand         string
	ImageURL      string
	KcalPer100    float64
	ProteinPer100 float64
	FatPer100     float64
	CarbsPer100   float64
	ServingGrams  *float64
}

type Client struct {
	http      *http.Client
	userAgent string
	baseURL   string
}

// New builds a client. contact is embedded in the User-Agent as OFF etiquette
// asks (so they can reach the app operator if needed).
func New(contact string) *Client {
	if contact == "" {
		contact = "https://kabanos.app"
	}
	return &Client{
		http:      &http.Client{Timeout: 8 * time.Second},
		userAgent: fmt.Sprintf("Kabanos/1.0 (%s)", contact),
		baseURL:   "https://world.openfoodfacts.org",
	}
}

// offResponse is the slice of the OFF v2 response we care about.
type offResponse struct {
	Status  int `json:"status"`
	Product struct {
		ProductName     string         `json:"product_name"`
		ProductNameRU   string         `json:"product_name_ru"`
		GenericName     string         `json:"generic_name"`
		Brands          string         `json:"brands"`
		ImageFrontSmall string         `json:"image_front_small_url"`
		ImageURL        string         `json:"image_url"`
		ServingQty      any            `json:"serving_quantity"` // OFF sends number or string
		Nutriments      map[string]any `json:"nutriments"`
	} `json:"product"`
}

// Lookup fetches and normalizes a product by barcode.
func (c *Client) Lookup(ctx context.Context, barcode string) (*Product, error) {
	url := c.baseURL + "/api/v2/product/" + barcode +
		".json?fields=product_name,product_name_ru,generic_name,brands,image_front_small_url,image_url,serving_quantity,nutriments"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openfoodfacts: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var r offResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("openfoodfacts: decode: %w", err)
	}
	if r.Status != 1 {
		return nil, ErrNotFound
	}
	return normalize(barcode, r), nil
}

func normalize(barcode string, r offResponse) *Product {
	p := &Product{Barcode: barcode, Brand: strings.TrimSpace(firstLine(r.Product.Brands))}

	name := firstNonEmpty(r.Product.ProductNameRU, r.Product.ProductName, r.Product.GenericName)
	p.Name = strings.TrimSpace(name)
	if p.Name == "" {
		p.Name = "Товар " + barcode
	}

	p.ImageURL = firstNonEmpty(r.Product.ImageFrontSmall, r.Product.ImageURL)

	n := r.Product.Nutriments
	p.KcalPer100 = kcalFromNutriments(n)
	p.ProteinPer100 = round1(numField(n, "proteins_100g"))
	p.FatPer100 = round1(numField(n, "fat_100g"))
	p.CarbsPer100 = round1(numField(n, "carbohydrates_100g"))

	if g := toFloat(r.Product.ServingQty); g > 0 {
		gg := round1(g)
		p.ServingGrams = &gg
	}
	return p
}

// kcalFromNutriments prefers an explicit kcal value, else converts kJ.
func kcalFromNutriments(n map[string]any) float64 {
	if v := numField(n, "energy-kcal_100g"); v > 0 {
		return round1(v)
	}
	if kj := numField(n, "energy-kj_100g"); kj > 0 {
		return round1(kj / 4.184)
	}
	if kj := numField(n, "energy_100g"); kj > 0 { // OFF's generic energy is in kJ
		return round1(kj / 4.184)
	}
	return 0
}

func numField(n map[string]any, key string) float64 {
	if n == nil {
		return 0
	}
	return toFloat(n[key])
}

// toFloat coerces the number-or-string values OFF returns.
func toFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f
	default:
		return 0
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// firstLine returns the first comma-separated brand (OFF joins brands with ", ").
func firstLine(s string) string {
	if i := strings.IndexByte(s, ','); i >= 0 {
		return s[:i]
	}
	return s
}

func round1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}
