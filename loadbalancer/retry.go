package loadbalancer

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/go-kit/kit/endpoint"
)

// Retry yields an endpoint that takes endpoints from the load balancer.
// Invocations that return errors will be retried until they succeed, up to
// max times, or until the timeout is elapsed, whichever comes first.
func Retry(max int, timeout time.Duration, lb LoadBalancer) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var (
			newctx, cancel = context.WithTimeout(ctx, timeout)
			responses      = make(chan interface{}, 1)
			errs           = make(chan error, 1)
			a              = []string{}
		)
		defer cancel()
		for i := 1; i <= max; i++ {
			go func() {
				e, err := lb.Get()
				if err != nil {
					errs <- err
					return
				}
				response, err := e(newctx, request)
				if err != nil {
					errs <- err
					return
				}
				responses <- response
			}()

			select {
			case <-newctx.Done():
				return nil, newctx.Err()
			case response := <-responses:
				return response, nil
			case err := <-errs:
				a = append(a, err.Error())
				continue
			}
		}
		return nil, fmt.Errorf("retry attempts exceeded (%s)", strings.Join(a, "; "))
	}
}
