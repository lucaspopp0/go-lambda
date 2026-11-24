// Package eventwrappers wraps the AWS Lambda HTTP event types in common
// HttpRequest and HttpResponse interfaces, so handler code can be written once
// and used with API Gateway v2 and Lambda Function URL events.
//
// Every wrapper can be unwrapped back to the original event type, either with
// the Unwrap method or the generic Unwrap/UnwrapRequest/UnwrapResponse helpers.
package eventwrappers
