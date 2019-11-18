// Code generated by private/model/cli/gen-api/main.go. DO NOT EDIT.

package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/internal/awsutil"
)

// Please also see https://docs.aws.amazon.com/goto/WebAPI/ec2-2016-11-15/ImportClientVpnClientCertificateRevocationListRequest
type ImportClientVpnClientCertificateRevocationListInput struct {
	_ struct{} `type:"structure"`

	// The client certificate revocation list file. For more information, see Generate
	// a Client Certificate Revocation List (https://docs.aws.amazon.com/vpn/latest/clientvpn-admin/cvpn-working-certificates.html#cvpn-working-certificates-generate)
	// in the AWS Client VPN Administrator Guide.
	//
	// CertificateRevocationList is a required field
	CertificateRevocationList *string `type:"string" required:"true"`

	// The ID of the Client VPN endpoint to which the client certificate revocation
	// list applies.
	//
	// ClientVpnEndpointId is a required field
	ClientVpnEndpointId *string `type:"string" required:"true"`

	// Checks whether you have the required permissions for the action, without
	// actually making the request, and provides an error response. If you have
	// the required permissions, the error response is DryRunOperation. Otherwise,
	// it is UnauthorizedOperation.
	DryRun *bool `type:"boolean"`
}

// String returns the string representation
func (s ImportClientVpnClientCertificateRevocationListInput) String() string {
	return awsutil.Prettify(s)
}

// Validate inspects the fields of the type to determine if they are valid.
func (s *ImportClientVpnClientCertificateRevocationListInput) Validate() error {
	invalidParams := aws.ErrInvalidParams{Context: "ImportClientVpnClientCertificateRevocationListInput"}

	if s.CertificateRevocationList == nil {
		invalidParams.Add(aws.NewErrParamRequired("CertificateRevocationList"))
	}

	if s.ClientVpnEndpointId == nil {
		invalidParams.Add(aws.NewErrParamRequired("ClientVpnEndpointId"))
	}

	if invalidParams.Len() > 0 {
		return invalidParams
	}
	return nil
}

// Please also see https://docs.aws.amazon.com/goto/WebAPI/ec2-2016-11-15/ImportClientVpnClientCertificateRevocationListResult
type ImportClientVpnClientCertificateRevocationListOutput struct {
	_ struct{} `type:"structure"`

	// Returns true if the request succeeds; otherwise, it returns an error.
	Return *bool `locationName:"return" type:"boolean"`
}

// String returns the string representation
func (s ImportClientVpnClientCertificateRevocationListOutput) String() string {
	return awsutil.Prettify(s)
}

const opImportClientVpnClientCertificateRevocationList = "ImportClientVpnClientCertificateRevocationList"

// ImportClientVpnClientCertificateRevocationListRequest returns a request value for making API operation for
// Amazon Elastic Compute Cloud.
//
// Uploads a client certificate revocation list to the specified Client VPN
// endpoint. Uploading a client certificate revocation list overwrites the existing
// client certificate revocation list.
//
// Uploading a client certificate revocation list resets existing client connections.
//
//    // Example sending a request using ImportClientVpnClientCertificateRevocationListRequest.
//    req := client.ImportClientVpnClientCertificateRevocationListRequest(params)
//    resp, err := req.Send(context.TODO())
//    if err == nil {
//        fmt.Println(resp)
//    }
//
// Please also see https://docs.aws.amazon.com/goto/WebAPI/ec2-2016-11-15/ImportClientVpnClientCertificateRevocationList
func (c *Client) ImportClientVpnClientCertificateRevocationListRequest(input *ImportClientVpnClientCertificateRevocationListInput) ImportClientVpnClientCertificateRevocationListRequest {
	op := &aws.Operation{
		Name:       opImportClientVpnClientCertificateRevocationList,
		HTTPMethod: "POST",
		HTTPPath:   "/",
	}

	if input == nil {
		input = &ImportClientVpnClientCertificateRevocationListInput{}
	}

	req := c.newRequest(op, input, &ImportClientVpnClientCertificateRevocationListOutput{})
	return ImportClientVpnClientCertificateRevocationListRequest{Request: req, Input: input, Copy: c.ImportClientVpnClientCertificateRevocationListRequest}
}

// ImportClientVpnClientCertificateRevocationListRequest is the request type for the
// ImportClientVpnClientCertificateRevocationList API operation.
type ImportClientVpnClientCertificateRevocationListRequest struct {
	*aws.Request
	Input *ImportClientVpnClientCertificateRevocationListInput
	Copy  func(*ImportClientVpnClientCertificateRevocationListInput) ImportClientVpnClientCertificateRevocationListRequest
}

// Send marshals and sends the ImportClientVpnClientCertificateRevocationList API request.
func (r ImportClientVpnClientCertificateRevocationListRequest) Send(ctx context.Context) (*ImportClientVpnClientCertificateRevocationListResponse, error) {
	r.Request.SetContext(ctx)
	err := r.Request.Send()
	if err != nil {
		return nil, err
	}

	resp := &ImportClientVpnClientCertificateRevocationListResponse{
		ImportClientVpnClientCertificateRevocationListOutput: r.Request.Data.(*ImportClientVpnClientCertificateRevocationListOutput),
		response: &aws.Response{Request: r.Request},
	}

	return resp, nil
}

// ImportClientVpnClientCertificateRevocationListResponse is the response type for the
// ImportClientVpnClientCertificateRevocationList API operation.
type ImportClientVpnClientCertificateRevocationListResponse struct {
	*ImportClientVpnClientCertificateRevocationListOutput

	response *aws.Response
}

// SDKResponseMetdata returns the response metadata for the
// ImportClientVpnClientCertificateRevocationList request.
func (r *ImportClientVpnClientCertificateRevocationListResponse) SDKResponseMetdata() *aws.Response {
	return r.response
}
