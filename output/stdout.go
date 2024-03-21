package output

import "log/slog"

type stdout struct{}

func (r *stdout) Output(projectID uint64, taskData [][]byte, proof []byte) (string, error) {
	slog.Info("stdout", "proof", string(proof))
	return "", nil
}

func newStdout() Output {
	return &stdout{}
}
