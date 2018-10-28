package main

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
)

var scSvc *servicecatalog.ServiceCatalog
var cfSvc *cloudformation.CloudFormation

//ResponseURL returns the cost estamate URL.
type responseURL struct {
	URL string `json:"url"`
}

//ResponseError returns an error response
type responseError struct {
	Error string `json:"error"`
}

func init() {
	//Open an AWS session.
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		log.Fatal("Sesson error: ", err)
	}

	//Create a service from the session
	scSvc = servicecatalog.New(sess)
	cfSvc = cloudformation.New(sess)
}

//Handler for the Lambda request
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var par []*cloudformation.Parameter

	byt := []byte(request.Body)

	proInput := servicecatalog.ProvisionProductInput{}

	//Check the json input and return an error if malformed.
	if err := json.Unmarshal(byt, &proInput); err != nil {
		log.Printf("error: %v", err)

		eres := &responseError{
			Error: "Invalid json",
		}

		econtent, _ := json.Marshal(eres)

		return events.APIGatewayProxyResponse{
			Body:       string(econtent),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
		}, nil

	}

	ap := servicecatalog.DescribeProvisioningArtifactInput{
		ProductId:              proInput.ProductId,
		ProvisioningArtifactId: proInput.ProvisioningArtifactId,
		Verbose:                aws.Bool(true),
	}

	//Describe the product and get the Cloudformation.
	out, err := scSvc.DescribeProvisioningArtifact(&ap)

	if err != nil {
		log.Printf("error: %v", err)
		eres := &responseError{
			Error: err.Error(),
		}

		econtent, _ := json.Marshal(eres)

		return events.APIGatewayProxyResponse{
			Body:       string(econtent),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
		}, nil
	}

	for _, pp := range proInput.ProvisioningParameters {
		par = append(par, &cloudformation.Parameter{

			ParameterKey:   pp.Key,
			ParameterValue: pp.Value,
		})
	}

	//Make request to obtain the cost of the deployment.
	cfOut, err := cfSvc.EstimateTemplateCost(&cloudformation.EstimateTemplateCostInput{
		Parameters:   par,
		TemplateBody: out.Info["CloudFormationTemplate"],
	})

	if err != nil {
		log.Printf("error: %v", err)
		eres := &responseError{
			Error: "Unable to get estamate",
		}

		econtent, _ := json.Marshal(eres)

		return events.APIGatewayProxyResponse{
			Body:       string(econtent),
			Headers:    map[string]string{"Content-Type": "application/json"},
			StatusCode: 400,
		}, nil
	}

	log.Printf("Estamate URL: %v", *cfOut.Url)
	res := &responseURL{
		URL: *cfOut.Url,
	}

	content, err := json.Marshal(res)

	//Return the URL.
	return events.APIGatewayProxyResponse{
		Body:       string(content),
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(Handler)
}
