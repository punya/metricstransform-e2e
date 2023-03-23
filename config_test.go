package configsandbox

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/efficientgo/core/testutil"
	"github.com/efficientgo/e2e"
	"github.com/otiai10/copy"
)

//go:embed config.yaml
var collectorConfig string

func makeTarget(t *testing.T, app string, env *e2e.DockerEnvironment, labelValue string, metricValue float64) e2e.Runnable {
	return env.Runnable("target-" + labelValue).
		WithPorts(map[string]int{
			"metrics": 8080,
		}).
		Init(e2e.StartOptions{
			Command: e2e.NewCommand(path.Join(app, "run.sh"),
				"-label-value", labelValue,
				"-metric-value", fmt.Sprintf("%f", metricValue),
			),
			Image: "golang:1.20",
			// Readiness: e2e.NewHTTPReadinessProbe("metrics", "/metrics", 200, 204),
		})
}

func makeOtelCollector(t *testing.T, env *e2e.DockerEnvironment, targets ...string) e2e.Runnable {
	scrapeTargets := strings.Join(targets, ",")

	configFile := path.Join(env.SharedDir(), "config.yaml")
	testutil.Ok(t, os.WriteFile(
		configFile,
		[]byte(fmt.Sprintf(collectorConfig, scrapeTargets)),
		0644,
	))

	return env.Runnable("otelcol").
		WithPorts(map[string]int{
			"health":   13133,
			"metrics1": 4000,
			"metrics2": 4001,
		}).
		Init(e2e.StartOptions{
			Command: e2e.NewCommand("--config", configFile),
			Image:   "otel/opentelemetry-collector-contrib:0.74.0",
			// Readiness: e2e.NewHTTPReadinessProbe("health", "/", 200, 204),
		})
}

func TestAll(t *testing.T) {
	env, err := e2e.New()
	t.Cleanup(env.Close)
	testutil.Ok(t, err)

	app := path.Join(env.SharedDir(), "app")

	os.Chmod(env.SharedDir(), 0755)
	copy.Copy("testapp", app)

	target1 := makeTarget(t, app, env, "ham", 10)
	target2 := makeTarget(t, app, env, "cheese", 20)
	target3 := makeTarget(t, app, env, "eggs", 40)

	otelcol := makeOtelCollector(t, env,
		target1.InternalEndpoint("metrics"),
		target2.InternalEndpoint("metrics"),
		target3.InternalEndpoint("metrics"),
	)

	testutil.Ok(t, e2e.StartAndWaitReady(target1, target2, target3))
	testutil.Ok(t, e2e.StartAndWaitReady(otelcol))

	for {
		time.Sleep(10 * time.Second)
		res, err := http.Get("http://" + otelcol.Endpoint("metrics2") + "/metrics")
		if err != nil {
			t.Log(err)
			continue
		}
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Log(err)
			continue
		}
		_ = res.Body.Close()
		t.Log(string(data))
	}
}
