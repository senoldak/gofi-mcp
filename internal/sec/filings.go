package sec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Filing struct {
	Form    string `json:"form"`
	FiledOn string `json:"filedOn"`
	Period  string `json:"period"`
	URL     string `json:"url"`
}

func (c *Client) Filings(ctx context.Context, ticker string) ([]Filing, error) {
	cik, err := c.lookupCIK(ctx, ticker)
	if err != nil {
		return nil, err
	}
	body, err := c.inner.Get(ctx, "/submissions/CIK"+cik+".json")
	if err != nil {
		return nil, fmt.Errorf("sec submissions: %w", err)
	}
	var doc struct {
		Filings struct {
			Recent struct {
				Form            []string `json:"form"`
				FilingDate      []string `json:"filingDate"`
				ReportDate      []string `json:"reportDate"`
				AccessionNumber []string `json:"accessionNumber"`
			} `json:"recent"`
		} `json:"filings"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("sec submissions decode: %w", err)
	}

	recent := doc.Filings.Recent
	n := len(recent.Form)
	if n > 10 {
		n = 10
	}
	cikNoZeros := strings.TrimLeft(cik, "0")
	out := make([]Filing, 0, n)
	for i := 0; i < n; i++ {
		accn := ""
		if i < len(recent.AccessionNumber) {
			accn = strings.ReplaceAll(recent.AccessionNumber[i], "-", "")
		}
		f := Filing{
			Form:    recent.Form[i],
			FiledOn: "", Period: "",
			URL: "https://www.sec.gov/Archives/edgar/data/" + cikNoZeros + "/" + accn,
		}
		if i < len(recent.FilingDate) {
			f.FiledOn = recent.FilingDate[i]
		}
		if i < len(recent.ReportDate) {
			f.Period = recent.ReportDate[i]
		}
		out = append(out, f)
	}
	return out, nil
}
