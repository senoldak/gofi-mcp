package sec

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

type Period struct {
	FiscalYear        int     `json:"fiscalYear"`
	FiscalPeriod      string  `json:"fiscalPeriod"`
	Revenue           float64 `json:"revenue"`
	NetIncome         float64 `json:"netIncome"`
	EPS               float64 `json:"eps"`
	TotalAssets       float64 `json:"totalAssets"`
	TotalLiabilities  float64 `json:"totalLiabilities"`
	OperatingCashFlow float64 `json:"operatingCashFlow"`
}

type Financials struct {
	Ticker  string   `json:"ticker"`
	Periods []Period `json:"periods"`
}

type usGaapField struct {
	tags []string
	unit string
	set  func(*Period, float64)
}

// usGaapFields maps SEC US-GAAP tags to Period fields. Multiple tags are
// tried in order because filers differ: e.g. Apple reports revenue under
// RevenueFromContractWithCustomerExcludingAssessedTax while others still use
// Revenues, and liabilities under Liabilities instead of TotalLiabilities.
var usGaapFields = []usGaapField{
	{tags: []string{"RevenueFromContractWithCustomerExcludingAssessedTax", "Revenues"}, unit: "USD", set: func(p *Period, v float64) { p.Revenue = v }},
	{tags: []string{"NetIncomeLoss"}, unit: "USD", set: func(p *Period, v float64) { p.NetIncome = v }},
	{tags: []string{"EarningsPerShareBasic"}, unit: "USD/shares", set: func(p *Period, v float64) { p.EPS = v }},
	{tags: []string{"Assets"}, unit: "USD", set: func(p *Period, v float64) { p.TotalAssets = v }},
	{tags: []string{"Liabilities", "TotalLiabilities"}, unit: "USD", set: func(p *Period, v float64) { p.TotalLiabilities = v }},
	{tags: []string{"NetCashProvidedByUsedInOperatingActivities"}, unit: "USD", set: func(p *Period, v float64) { p.OperatingCashFlow = v }},
}

func normalizeFinancials(cik string, raw []byte) (Financials, error) {
	var doc struct {
		Facts struct {
			UsGAAP map[string]struct {
				Units map[string][]struct {
					Val  float64 `json:"val"`
					FY   int     `json:"fy"`
					FP   string  `json:"fp"`
					Form string  `json:"form"`
				} `json:"units"`
			} `json:"us-gaap"`
		} `json:"facts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Financials{}, fmt.Errorf("sec companyfacts decode: %w", err)
	}

	periods := map[string]Period{}
	for _, field := range usGaapFields {
		for _, tag := range field.tags {
			gaap, ok := doc.Facts.UsGAAP[tag]
			if !ok {
				continue
			}
			recs, ok := gaap.Units[field.unit]
			if !ok {
				for _, u := range gaap.Units {
					recs = u
					break
				}
			}
			for _, rec := range recs {
				if rec.Form != "10-K" || rec.FP != "FY" {
					continue
				}
				key := strconv.Itoa(rec.FY) + ":" + rec.FP
				p, ok := periods[key]
				if !ok {
					p = Period{FiscalYear: rec.FY, FiscalPeriod: rec.FP}
				}
				field.set(&p, rec.Val)
				periods[key] = p
			}
			break
		}
	}

	out := make([]Period, 0, len(periods))
	for _, p := range periods {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FiscalYear > out[j].FiscalYear })
	if len(out) > 5 {
		out = out[:5]
	}
	return Financials{Ticker: cik, Periods: out}, nil
}
