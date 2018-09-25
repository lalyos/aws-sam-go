package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	gitter "github.com/sromku/go-gitter"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)
var api *gitter.Gitter

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	resp, err := http.Get(DefaultHTTPGetAddress)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if resp.StatusCode != 200 {
		return events.APIGatewayProxyResponse{}, ErrNon200Response
	}

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if len(ip) == 0 {
		return events.APIGatewayProxyResponse{}, ErrNoIP
	}

	ins, err := describeInstances()
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	msg := fmt.Sprintf("```\n%s\n```", ins)
	err = gitterMsg(msg)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Hello, %v", string(ip)),
		StatusCode: 200,
	}, nil
}
func getIp(ins *ec2.Instance) string {
	if ins == nil {
		return "n/a"
	}
	ip := ins.PublicIpAddress
	if ip == nil {
		return "n/a"
	}
	return *ip
}

func gitterMsg(msg string) error {
	token := os.Getenv("GITTER_TOKEN")
	if token == "" {
		return fmt.Errorf("GITTER_TOKEN missing, see: https://developer.gitter.im/apps")
	}
	room := os.Getenv("GITTER_ROOM")
	if room == "" {
		return fmt.Errorf("GITTER_ROOM missing")
	}

	api = gitter.New(token)
	_, err := api.SendMessage(room, msg)
	log.Printf("--> msg sent successfully: %#v\n", msg)
	return err
}

func describeInstances() (string, error) {
	reg := os.Getenv("AWS_DEFAULT_REGION")
	if reg == "" {
		return "", fmt.Errorf("Missing: AWS_DEFAULT_REGION env var")
	}
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(reg)}))

	inp := new(ec2.DescribeInstancesInput)
	resp, err := svc.DescribeInstances(inp)
	if err != nil {
		return "", fmt.Errorf("ec2.DescribeInstances failed: %v", err)
	}
	w := new(strings.Builder)
	for _, r := range resp.Reservations {
		for _, ins := range r.Instances {

			fmt.Fprintf(w, "id: %v, ip: %v\n", *ins.InstanceId, getIp(ins))

		}
	}
	return w.String(), nil
}

func sendTest() {
	ins, err := describeInstances()
	if err != nil {
		//return events.APIGatewayProxyResponse{}, err
		panic(err)
	}
	msg := fmt.Sprintf("```\n%s\n```", ins)
	err = gitterMsg(msg)
	if err != nil {
		panic(err)
	}
}
func main() {
	//sendTest()
	lambda.Start(handler)
}
