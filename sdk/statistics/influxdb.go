package statistics

import (
	"crypto/tls"
	"fmt"
	"github.com/configcat/configcat-proxy/config"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"time"
)

type influxReporter struct {
	client influxdb2.Client
	writer api.WriteAPI
}

func NewInfluxDbReporter(conf *config.InfluxDbConfig) Reporter {
	options := influxdb2.DefaultOptions().SetBatchSize(20)
	if conf.Tls.Enabled {
		t := &tls.Config{
			MinVersion: conf.Tls.GetVersion(),
			ServerName: conf.Tls.ServerName,
		}
		for _, c := range conf.Tls.Certificates {
			if cert, err := tls.LoadX509KeyPair(c.Cert, c.Key); err == nil {
				t.Certificates = append(t.Certificates, cert)
			}
		}
		options = options.SetTLSConfig(t)
	}
	client := influxdb2.NewClientWithOptions(conf.Url, conf.AuthToken, influxdb2.DefaultOptions().SetBatchSize(20))
	writer := client.WriteAPI(conf.Organization, conf.Bucket)
	return &influxReporter{
		client: client,
		writer: writer,
	}
}

func (r *influxReporter) ReportEvaluation(envId string, flagKey string, value interface{}, attrs map[string]string) {
	point := influxdb2.NewPointWithMeasurement(envId).
		AddTag("flag_key", flagKey).
		AddTag("flag_eval_value", fmt.Sprintf("%v", value)).
		AddField("value", 1).
		SetTime(time.Now().UTC())

	if attrs != nil {
		for key, val := range attrs {
			point.AddTag("user_"+key, val)
		}
	}
	r.writer.WritePoint(point)
}

func (r *influxReporter) Close() {
	r.client.Close()
}
