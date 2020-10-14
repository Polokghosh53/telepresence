package client

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/datawire/telepresence2/pkg/rpc"
)

type interceptState struct {
	cc  rpc.ConnectorClient
	ir  *rpc.InterceptRequest
	out io.Writer
}

func newInterceptState(cs rpc.ConnectorClient, ir *rpc.InterceptRequest, out io.Writer) *interceptState {
	return &interceptState{cc: cs, ir: ir, out: out}
}

func (is *interceptState) EnsureState() (bool, error) {
	r, err := is.cc.AddIntercept(context.Background(), is.ir)
	if err != nil {
		return false, err
	}
	switch r.Error {
	case rpc.InterceptError_InterceptOk:
		fmt.Fprintf(is.out, "Using deployment %s in namespace %s\n", is.ir.Deployment, r.Text)

		if r.PreviewURL != "" {
			fmt.Fprintf(is.out, "Share a preview of your changes with anyone by visiting\n  %s\n", r.PreviewURL)
		}
		return true, nil
	case rpc.InterceptError_AlreadyExists:
		fmt.Fprintln(is.out, interceptMessage(r.Error, r.Text))
		return false, nil
	case rpc.InterceptError_NoConnection:
		return false, connectorIsNotRunning
	}
	return false, errors.New(interceptMessage(r.Error, r.Text))
}

func (is *interceptState) DeactivateState() error {
	name := strings.TrimSpace(is.ir.Name)
	var r *rpc.InterceptResponse
	var err error
	r, err = is.cc.RemoveIntercept(context.Background(), &rpc.RemoveInterceptRequest{Name: name})
	if err != nil {
		return err
	}
	if r.Error != rpc.InterceptError_InterceptOk {
		return errors.New(interceptMessage(r.Error, r.Text))
	}
	return nil
}
