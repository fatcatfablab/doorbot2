package sender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fatcatfablab/doorbot2/types"
)

const (
	doordDateFmt = "01/02/2006"
	doordTimeFmt = time.TimeOnly
)

type DoordSender struct {
	client *http.Client
	url    *url.URL
}

func NewDoord(url *url.URL) *DoordSender {
	return &DoordSender{url: url, client: &http.Client{Timeout: 2 * time.Second}}
}

func (d *DoordSender) Post(ctx context.Context, stats types.Stats) error {
	var b bytes.Buffer
	fmt.Fprintf(
		&b,
		"%s,%s,%s,%d",
		stats.Last.Format(doordDateFmt),
		stats.Last.Format(doordTimeFmt),
		stats.Name,
		1,
	)
	req := &http.Request{
		Method:        http.MethodPost,
		URL:           d.url,
		Body:          io.NopCloser(&b),
		ContentLength: int64(b.Len()),
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending doord request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("doord request returned: %s", resp.Status)
	}

	return nil
}
