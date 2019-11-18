// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
	"github.com/aws/aws-sdk-go-v2/private/protocol"
	"github.com/aws/aws-sdk-go-v2/private/protocol/query"
)

// Please also see https://docs.aws.amazon.com/goto/WebAPI/sqs-2012-11-05/UntagQueueRequest
type UntagQueueInput struct {
	_ struct{} `type:"structure"`

	// The URL of the queue.
	//
	// QueueUrl is a required field
	QueueUrl *string `type:"string" required:"true"`

	// The list of tags to be removed from the specified queue.
	//
	// TagKeys is a required field
	TagKeys []string `locationNameList:"TagKey" type:"list" flattened:"true" required:"true"`
}

// String returns the string representation
func (s UntagQueueInput) String() string {
	return awsutil.Prettify(s)
}

// Validate inspects the fields of the type to determine if they are valid.
func (s *UntagQueueInput) Validate() error {
	invalidParams := aws.ErrInvalidParams{Context: "UntagQueueInput"}

	if s.QueueUrl == nil {
		invalidParams.Add(aws.NewErrParamRequired("QueueUrl"))
	}

	if s.TagKeys == nil {
		invalidParams.Add(aws.NewErrParamRequired("TagKeys"))
	}

	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

// Please also see https://docs.aws.amazon.com/goto/WebAPI/sqs-2012-11-05/UntagQueueOutput
type UntagQueueOutput struct {
	_ struct{} `type:"structure"`
}

// String returns the string representation
func (s UntagQueueOutput) String() string {
	return awsutil.Prettify(s)
}

const opUntagQueue = "UntagQueue"

// UntagQueueRequest returns a request value for making API operation for
// Amazon Simple Queue Service.
//
// Remove cost allocation tags from the specified Amazon SQS queue. For an overview,
// see Tagging Your Amazon SQS Queues (http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-queue-tags.html)
// in the Amazon Simple Queue Service Developer Guide.
//
// When you use queue tags, keep the following guidelines in mind:
//
//    * Adding more than 50 tags to a queue isn't recommended.
//
//    * Tags don't have any semantic meaning. Amazon SQS interprets tags as
//    character strings.
//
//    * Tags are case-sensitive.
//
//    * A new tag with a key identical to that of an existing tag overwrites
//    the existing tag.
//
//    * Tagging actions are limited to 5 TPS per AWS account. If your application
//    requires a higher throughput, file a technical support request (https://console.aws.amazon.com/support/home#/case/create?issueType=technical).
//
// For a full list of tag restrictions, see Limits Related to Queues (http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-limits.html#limits-queues)
// in the Amazon Simple Queue Service Developer Guide.
//
// Cross-account permissions don't apply to this action. For more information,
// see see Grant Cross-Account Permissions to a Role and a User Name (http://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-customer-managed-policy-examples.html#grant-cross-account-permissions-to-role-and-user-name)
// in the Amazon Simple Queue Service Developer Guide.
//
//    // Example sending a request using UntagQueueRequest.
//    req := client.UntagQueueRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/sqs-2012-11-05/UntagQueue
func (c *Client) UntagQueueRequest(input *UntagQueueInput) UntagQueueRequest {
	op := &aws.Operation{
		Name:       opUntagQueue,
		HTTPMethod: "POST",
		HTTPPath:   "/",
	}

	if input == nil {
		input = &UntagQueueInput{}
	}

	req := c.newRequest(op, input, &UntagQueueOutput{})
	req.Handlers.Unmarshal.Remove(query.UnmarshalHandler)
	req.Handlers.Unmarshal.PushBackNamed(protocol.UnmarshalDiscardBodyHandler)
	return UntagQueueRequest{Request: req, Input: input, Copy: c.UntagQueueRequest}
}

// UntagQueueRequest is the request type for the
// UntagQueue API operation.
type UntagQueueRequest struct {
	*aws.Request
	Input *UntagQueueInput
	Copy  func(*UntagQueueInput) UntagQueueRequest
}

// Send marshals and sends the UntagQueue API request.
func (r UntagQueueRequest) Send(ctx context.Context) (*UntagQueueResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &UntagQueueResponse{
		UntagQueueOutput: r.Request.Data.(*UntagQueueOutput),
		response:         &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// UntagQueueResponse is the response type for the
// UntagQueue API operation.
type UntagQueueResponse struct {
	*UntagQueueOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// UntagQueue request.
func (r *UntagQueueResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
