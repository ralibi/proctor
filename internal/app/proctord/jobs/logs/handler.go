package logs

import (
	"bufio"
	"github.com/getsentry/raven-go"
	"io"
	"net/http"
	"proctor/internal/app/service/infra/config"
	"proctor/internal/app/service/infra/kubernetes"
	_logger "proctor/internal/app/service/infra/logger"
	"strings"
	"time"

	"proctor/internal/pkg/constant"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  config.LogsStreamReadBufferSize(),
	WriteBufferSize: config.LogsStreamWriteBufferSize(),
}

type logger struct {
	kubeClient kubernetes.KubernetesClient
}

type Logger interface {
	Stream() http.HandlerFunc
}

func NewLogger(kubeClient kubernetes.KubernetesClient) Logger {
	return &logger{
		kubeClient: kubeClient,
	}
}

func CloseWebSocket(message string, conn *websocket.Conn) {
	err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, message))
	if err != nil {
		_logger.Error("Error closing connection with client after logs are read")
		raven.CaptureError(err, nil)
	}
	return
}

func (l *logger) Stream() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			_logger.Error("Error upgrading connection to websocket protocol: ", err)
			raven.CaptureError(err, nil)

			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(constant.ClientError))
			return
		}
		defer conn.Close()

		jobName := strings.TrimLeft(req.URL.RawQuery, "job_name=")
		if jobName == "" {
			_logger.Error("No job name provided as part of URL: ", req.URL.RawQuery)
			CloseWebSocket("No job name provided while requesting for logs", conn)
			return
		}

		waitTime := config.KubePodsListWaitTime() * time.Second
		logStream, err := l.kubeClient.StreamJobLogs(jobName, waitTime)
		if err != nil {
			_logger.Error("Error streaming logs from kube client: ", err)
			raven.CaptureError(err, map[string]string{"job_name": jobName})

			CloseWebSocket("Something went wrong", conn)
			return
		}
		defer logStream.Close()

		bufioReader := bufio.NewReader(logStream)

		for {
			jobLogSingleLine, _, err := bufioReader.ReadLine()
			if err != nil {
				if err == io.EOF {
					_logger.Debug("Finished streaming logs for job: ", jobName)
					CloseWebSocket("All logs are read", conn)
					return
				}

				_logger.Error("Error reading from reader: ", err.Error())
				raven.CaptureError(err, nil)

				CloseWebSocket("Something went wrong", conn)
				return
			}

			_logger.Debug("writing to web socket ", string(jobLogSingleLine[:]))
			err = conn.WriteMessage(websocket.TextMessage, jobLogSingleLine[:])
			if err != nil {
				_logger.Error("Error writing logs to client: ", err)
				raven.CaptureError(err, nil)

				CloseWebSocket("Something went wrong", conn)
				return
			}
		}
	}
}
