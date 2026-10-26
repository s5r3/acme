package order

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"example.com/acme/internal/feature/order/views"
)

// FormRows is how many item rows the "new order" form shows.
const FormRows = 3

// CreateOrderForm is the web form for a new order. Item rows arrive as
// repeated fields (sku, name, quantity, price), one value per row.
type CreateOrderForm struct {
	Note string
	Rows []views.ItemRow
}

// parseCreateOrderForm reads the posted form. It never fails on content;
// content problems are reported by toInput.
func parseCreateOrderForm(r *http.Request) (CreateOrderForm, error) {
	if err := r.ParseForm(); err != nil {
		return CreateOrderForm{}, err
	}
	f := CreateOrderForm{Note: strings.TrimSpace(r.PostForm.Get("note"))}
	skus, names, qtys, prices := r.PostForm["sku"], r.PostForm["name"], r.PostForm["quantity"], r.PostForm["price"]
	for i := range FormRows {
		f.Rows = append(f.Rows, views.ItemRow{SKU: at(skus, i), Name: at(names, i), Quantity: at(qtys, i), Price: at(prices, i)})
	}
	return f, nil
}

// toInput converts the form to the service input. Blank rows are skipped;
// rowOf[i] is the form row of service item i, used to show service errors
// ("items[0].sku") next to the right row. Unparseable numbers become field
// errors keyed by form row.
func (f CreateOrderForm) toInput() (in CreateOrderInput, rowOf []int, errs map[string]string) {
	in = CreateOrderInput{Note: f.Note}
	errs = map[string]string{}
	if len(f.Note) > 500 {
		errs["note"] = "must be at most 500 characters"
	}
	for i, row := range f.Rows {
		if row.Blank() {
			continue
		}
		qty, err := strconv.Atoi(strings.TrimSpace(row.Quantity))
		if err != nil || qty <= 0 {
			errs[fmt.Sprintf("items[%d].quantity", i)] = "must be a whole number above 0"
			continue
		}
		cents, err := parseMoney(row.Price)
		if err != nil {
			errs[fmt.Sprintf("items[%d].price", i)] = "must be an amount like 12.50"
			continue
		}
		in.Items = append(in.Items, ItemInput{SKU: row.SKU, Name: row.Name, Quantity: qty, UnitPriceCents: cents})
		rowOf = append(rowOf, i)
	}
	if len(errs) == 0 {
		errs = nil
	}
	return in, rowOf, errs
}

// remapItemErrors rewrites service keys "items[n].x" to form-row keys.
func remapItemErrors(fields map[string]string, rowOf []int) map[string]string {
	out := make(map[string]string, len(fields))
	for k, v := range fields {
		var n int
		var rest string
		if _, err := fmt.Sscanf(k, "items[%d]%s", &n, &rest); err == nil && n < len(rowOf) {
			k = fmt.Sprintf("items[%d]%s", rowOf[n], rest)
		}
		out[k] = v
	}
	return out
}

// parseMoney turns "12.5" or "12.50" into 1250 cents.
func parseMoney(s string) (int64, error) {
	s = strings.TrimSpace(s)
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" || len(frac) > 2 {
		return 0, fmt.Errorf("bad amount %q", s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || w < 0 {
		return 0, fmt.Errorf("bad amount %q", s)
	}
	fr, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bad amount %q", s)
	}
	return w*100 + fr, nil
}

func at(vals []string, i int) string {
	if i < len(vals) {
		return strings.TrimSpace(vals[i])
	}
	return ""
}
