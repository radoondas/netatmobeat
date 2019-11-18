// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package elasticloadbalancingv2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
)

// Please also see https://docs.aws.amazon.com/goto/WebAPI/elasticloadbalancingv2-2015-12-01/DescribeListenersInput
type DescribeListenersInput struct {
	_ struct{} `type:"structure"`

	// The Amazon Resource Names (ARN) of the listeners.
	ListenerArns []string `type:"list"`

	// The Amazon Resource Name (ARN) of the load balancer.
	LoadBalancerArn *string `type:"string"`

	// The marker for the next set of results. (You received this marker from a
	// previous call.)
	Marker *string `type:"string"`

	// The maximum number of results to return with this call.
	PageSize *int64 `min:"1" type:"integer"`
}

// String returns the string representation
func (s DescribeListenersInput) String() string {
	return awsutil.Prettify(s)
}

// Validate inspects the fields of the type to determine if they are valid.
func (s *DescribeListenersInput) Validate() error {
	invalidParams := aws.ErrInvalidParams{Context: "DescribeListenersInput"}
	if s.PageSize != nil && *s.PageSize < 1 {
		invalidParams.Add(aws.NewErrParamMinValue("PageSize", 1))
	}

	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

// Please also see https://docs.aws.amazon.com/goto/WebAPI/elasticloadbalancingv2-2015-12-01/DescribeListenersOutput
type DescribeListenersOutput struct {
	_ struct{} `type:"structure"`

	// Information about the listeners.
	Listeners []Listener `type:"list"`

	// The marker to use when requesting the next set of results. If there are no
	// additional results, the string is empty.
	NextMarker *string `type:"string"`
}

// String returns the string representation
func (s DescribeListenersOutput) String() string {
	return awsutil.Prettify(s)
}

const opDescribeListeners = "DescribeListeners"

// DescribeListenersRequest returns a request value for making API operation for
// Elastic Load Balancing.
//
// Describes the specified listeners or the listeners for the specified Application
// Load Balancer or Network Load Balancer. You must specify either a load balancer
// or one or more listeners.
//
//    // Example sending a request using DescribeListenersRequest.
//    req := client.DescribeListenersRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/elasticloadbalancingv2-2015-12-01/DescribeListeners
func (c *Client) DescribeListenersRequest(input *DescribeListenersInput) DescribeListenersRequest {
	op := &aws.Operation{
		Name:       opDescribeListeners,
		HTTPMethod: "POST",
		HTTPPath:   "/",
		Paginator: &aws.Paginator{
			InputTokens:     []string{"Marker"},
			OutputTokens:    []string{"NextMarker"},
			LimitToken:      "",
			TruncationToken: "",
		},
	}

	if input == nil {
		input = &DescribeListenersInput{}
	}

	req := c.newRequest(op, input, &DescribeListenersOutput{})
	return DescribeListenersRequest{Request: req, Input: input, Copy: c.DescribeListenersRequest}
}

// DescribeListenersRequest is the request type for the
// DescribeListeners API operation.
type DescribeListenersRequest struct {
	*aws.Request
	Input *DescribeListenersInput
	Copy  func(*DescribeListenersInput) DescribeListenersRequest
}

// Send marshals and sends the DescribeListeners API request.
func (r DescribeListenersRequest) Send(ctx context.Context) (*DescribeListenersResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &DescribeListenersResponse{
		DescribeListenersOutput: r.Request.Data.(*DescribeListenersOutput),
		response:                &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// NewDescribeListenersRequestPaginator returns a paginator for DescribeListeners.
// Use Next method to get the next page, and CurrentPage to get the current
// response page from the paginator. Next will return false, if there are
// no more pages, or an error was encountered.
//
// Note: This operation can generate multiple requests to a service.
//
//   // Example iterating over pages.
//   req := client.DescribeListenersRequest(input)
//   p := elasticloadbalancingv2.NewDescribeListenersRequestPaginator(req)
//
//   for p.Next(context.TODO()) {
//       page := p.CurrentPage()
//   }
//
//   if err := p.Err(); err != nil {
//       return err
//   }
//
func NewDescribeListenersPaginator(req DescribeListenersRequest) DescribeListenersPaginator {
	return DescribeListenersPaginator{
		Pager: aws.Pager{
			NewRequest: func(ctx context.Context) (*aws.Request, error) {
				var inCpy *DescribeListenersInput
				if req.Input != nil {
					tmp := *req.Input
					inCpy = &tmp
				}

				newReq := req.Copy(inCpy)
				newReq.SetContext(ctx)
				return newReq.Request, nil
			},
		},
	}
}

// DescribeListenersPaginator is used to paginate the request. This can be done by
// calling Next and CurrentPage.
type DescribeListenersPaginator struct {
	aws.Pager
}

func (p *DescribeListenersPaginator) CurrentPage() *DescribeListenersOutput {
	return p.Pager.CurrentPage().(*DescribeListenersOutput)
}

// DescribeListenersResponse is the response type for the
// DescribeListeners API operation.
type DescribeListenersResponse struct {
	*DescribeListenersOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// DescribeListeners request.
func (r *DescribeListenersResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
