package main

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type WebNotifier struct {
	checkStates map[CheckInfo]CheckState
}

func NewWebNotifier(sm *SubroutineManager, listenAddress string) *WebNotifier {
	wn := &WebNotifier{
		checkStates: make(map[CheckInfo]CheckState),
	}

	r := chi.NewRouter()
	r.Get("/", wn.handleStatusOverview)
	r.Get("/health", wn.handleHealth)

	srv := &http.Server{
		Addr:    listenAddress,
		Handler: r,
	}
	sm.Add(2)
	go func() {
		defer sm.Done()
		zap.L().Info("Start web listener.", zap.String("listenAddress", listenAddress))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Cannot start web listener.", zap.String("listenAddress", listenAddress), zap.Error(err))
		}
	}()
	go func() {
		defer sm.Done()
		<-sm.Context().Done()
		shutdownCtx, _ := context.WithTimeout(sm.Context(), 10*time.Second)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			zap.L().Error("Cannot shutdown web listener.", zap.Error(err))
		} else {
			zap.L().Info("Web listener has been stopped.", zap.String("listenAddress", listenAddress))
		}
	}()

	return wn
}

func (wn *WebNotifier) handleStatusOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	if err := statusPageTemplate.Execute(w, wn.checkStates); err != nil {
		zap.L().Error("Cannot render status page.", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (wn *WebNotifier) handleHealth(w http.ResponseWriter, r *http.Request) {
	allOk := true
	for _, cs := range wn.checkStates {
		if !cs.Ok {
			allOk = false
		}
	}

	if allOk {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (wn *WebNotifier) Notify(info CheckInfo, state CheckState) error {
	wn.checkStates[info] = state
	return nil
}

var statusPageTemplate = template.Must(template.New("statusPageTemplate").Parse(`<html>
<head>
	<title>Server Status</title>
</head>
<body>

<table>
	<tr>
		<th>Check</th>
		<th>Status</th>
		<th>Reason</th>
	</tr>
{{ range $check, $status := . }}
	<tr style="background-color: {{ if $status.Ok }}green{{ else }}red{{ end }}">
		<td>{{ $check.CheckName }}</td>
		<td>{{ if $status.Ok }}OK{{ else }}Failed{{ end }}</td>
		<td>{{ $status.Reason }}</td>
	</tr>
{{ end }}
</table>

</body>
</html>
`))
