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

var usGaapTags = []string{
	"Revenues",
	"NetIncomeLoss",
	"EarningsPerShareBasic",
	"Assets",
	"TotalLiabilities",
	"NetCashProvidedByUsedInOperatingActivities",
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
	for _, tag := range usGaapTags {
		gaap, ok := doc.Facts.UsGAAP[tag]
		if !ok {
			continue
		}
		for _, rec := range gaap.Units["USD"] {
			if rec.Form != "10-K" || rec.FP != "FY" {
				continue
			}
			key := strconv.Itoa(rec.FY) + ":" + rec.FP
			p, ok := periods[key]
			if !ok {
				p = Period{FiscalYear: rec.FY, FiscalPeriod: rec.FP}
			}
			switch tag {
			case "Revenues":
				p.Revenue = rec.Val
			case "NetIncomeLoss":
				p.NetIncome = rec.Val
			case "EarningsPerShareBasic":
				p.EPS = rec.Val
			case "Assets":
				p.TotalAssets = rec.Val
			case "TotalLiabilities":
				p.TotalLiabilities = rec.Val
			case "NetCashProvidedByUsedInOperatingActivities":
				p.OperatingCashFlow = rec.Val
			}
			periods[key] = p
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
