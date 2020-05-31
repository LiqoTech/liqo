package federation_request_admission

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	Log = ctrl.Log.WithName("federation-webhook-admission")
)

func StartWebhook(certPath string) *WebhookServer {
	port := 8443
	return startTls(certPath, port)
}

func startTls(certPath string, port int) *WebhookServer {
	certFile := certPath + "/cert.pem"
	keyFile := certPath + "/key.pem"

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		Log.Error(err, err.Error())
		os.Exit(1)
	}

	whsvr := &WebhookServer{
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http Server and Server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", whsvr.serve)
	whsvr.Server.Handler = mux

	// start webhook Server in new routine
	go func() {
		if err := whsvr.Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			Log.Error(err, "Failed to listen and serve webhook Server: "+err.Error())
		}
	}()

	Log.Info("Server started")

	return whsvr
}
