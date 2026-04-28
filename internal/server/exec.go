package server

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/coder/websocket"
	"github.com/polds/k8s-pod-shell/internal/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type controlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func RunExecSession(ctx context.Context, conn *websocket.Conn, cs kubernetes.Interface, ns, pod, container string) error {
	// Build stream readers/writers.
	stdinR, stdinW := io.Pipe()
	defer func() { _ = stdinR.Close() }()

	resizeQueue := &resizeEventQueue{ch: make(chan remotecommand.TerminalSize, 8)}
	out := wsWriter{conn: conn}

	// Pump websocket input into stdin and resize queue.
	var readErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() { _ = stdinW.Close() }()
		for {
			_, payload, err := conn.Read(ctx)
			if err != nil {
				readErr = err
				return
			}
			typ, body, err := readFrameData(payload)
			if err != nil {
				readErr = err
				return
			}
			switch typ {
			case 0x00:
				if _, err := stdinW.Write(body); err != nil {
					readErr = err
					return
				}
			case 0x01:
				var msg controlMessage
				if err := json.Unmarshal(body, &msg); err != nil {
					continue
				}
				if msg.Type == "resize" {
					resizeQueue.push(remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows})
				}
			default:
				readErr = errUnsupportedControl
				return
			}
		}
	}()

	cfg, err := kube.RestConfig()
	if err != nil {
		return err
	}

	cmd, err := SelectShell(ctx, cs, ns, pod, container)
	if err != nil {
		return err
	}
	req := cs.CoreV1().RESTClient().Post().Resource("pods").Namespace(ns).Name(pod).SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdinR,
		Stdout:            out,
		Stderr:            out,
		Tty:               true,
		TerminalSizeQueue: resizeQueue,
	})
	_ = sendStatus(conn, map[string]any{"type": "exit", "code": 0})
	wg.Wait()
	if readErr != nil {
		return readErr
	}
	return err
}

func SelectShell(ctx context.Context, cs kubernetes.Interface, ns, pod, container string) ([]string, error) {
	shells := []string{"bash", "zsh", "ash", "sh"}
	// TODO: Proper probe via remote exec <shell> -c true.
	// For now, default to first shell for compatibility with distroless sidecars.
	_ = ctx
	_ = cs
	_ = ns
	_ = pod
	_ = container
	return []string{shells[0], "-il"}, nil
}

type resizeEventQueue struct {
	ch chan remotecommand.TerminalSize
}

func (q *resizeEventQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

func (q *resizeEventQueue) push(size remotecommand.TerminalSize) {
	select {
	case q.ch <- size:
	default:
	}
}
