package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

var apiURL string
var usr string
var pas string
var cli string
var grt string
var busID string
var svc *ssm.SSM

//The ResponseToken type stores the token that is returned during the auth flow.
type ResponseToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    string `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Issued       string `json:".issued"`
	Expires      string `json:".expires"`
}

//The Response type  stores the return response.
type Response struct {
	Message string `json:"message"`
}

//The BusiObject type stores information about the Cherwell BusiObject.
type BusiObject struct {
	BusObID     string `json:"busObId"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
}

//The Mainevent type stores information about the Cherwell event.
type Mainevent struct {
	BusObID string  `json:"busObId"`
	Fields  []field `json:"fields"`
	Persist bool    `json:"persist"`
}

//The field stores Cherwell field information.
type field struct {
	Dirty   bool   `json:"dirty"`
	FieldID string `json:"fieldId"`
	Value   string `json:"value"`
}

//ResponseError returns an error response
type responseError struct {
	Error string `json:"error"`
}

//complete the required Auth flow and store the token.
func init() {

	//Get and set the ENV.
	urlName := os.Getenv("URL")
	if urlName == "" {
		log.Fatal("$URL not set")
	}
	usrName := os.Getenv("USER")

	if usrName == "" {
		log.Fatal("$USERNAME not set")
	}
	pasName := os.Getenv("PASSWORD")

	if pasName == "" {
		log.Fatal("$PASSWORD not set")
	}
	cliName := os.Getenv("CLIENT_ID")

	if cliName == "" {
		log.Fatal("$CLIENT_ID not set")
	}
	grtName := os.Getenv("GRANT")

	if grtName == "" {
		log.Fatal("$GRANT not set")
	}

	//Open an AWS session.
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal("Sesson error: ", err)
	}

	//Create a service from the session
	svc = ssm.New(sess)

	sdIn := ssm.GetParametersInput{}
	parList := make([]string, 5)
	parList[0] = os.Getenv("URL")
	parList[1] = os.Getenv("USER")
	parList[2] = os.Getenv("PASSWORD")
	parList[3] = os.Getenv("CLIENT_ID")
	parList[4] = os.Getenv("GRANT")

	sdIn.SetNames(aws.StringSlice(parList))

	sdOut, err := svc.GetParameters(&sdIn)

	if err != nil {
		log.Fatal("Error getting parameter", err)
	}

	prs := make(map[string]string)

	for _, pr := range sdOut.Parameters {
		prs[*pr.Name] = *pr.Value
	}
	apiURL = prs[os.Getenv("URL")]
	usr = prs[os.Getenv("USER")]
	pas = prs[os.Getenv("PASSWORD")]
	cli = prs[os.Getenv("CLIENT_ID")]
	grt = prs[os.Getenv("GRANT")]

	log.Println("Endpoint:" + apiURL)

}

//Handler for the Lambda request
func Handler(request events.CloudWatchEvent) (Response, error) {

	log.Print("complete request")

	//Build the URL.
	data := url.Values{}
	data.Add("client_id", cli)
	data.Add("username", usr)
	data.Add("password", pas)
	data.Add("grant_type", grt)

	u, err := url.ParseRequestURI(apiURL)
	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	u.Path = "/CherwellAPI/token"

	fmt.Printf("Server URL: %v", u.String())
	client := &http.Client{}

	//Build the The Auth request.
	r, err := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))

	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	//Make the Auth Request
	resp, err := client.Do(r)

	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}
	defer resp.Body.Close()

	rt := ResponseToken{}

	rtData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	json.Unmarshal(rtData, &rt)

	rd, err := request.Detail.MarshalJSON()

	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	//Build the BusiObject slice.
	busID = "943e3fc2a8dab7f22f1d174bdf8e0c7010681362ec"
	f1 := field{
		Dirty:   true,
		FieldID: "BO:943e3fc2a8dab7f22f1d174bdf8e0c7010681362ec,FI:943e3fc42d204079d03a2947f9840e4c6f45884f35",
		Value:   string(rd),
	}

	var f1s []field
	f1s = append(f1s, f1)

	iD1 := Mainevent{
		BusObID: busID,
		Fields:  f1s,
		Persist: true,
	}

	jsonStr, err := json.Marshal(iD1)
	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}
	log.Printf("Cherwell request: %v", iD1)

	//Make a request and send event info to Cherwell.
	freq, err := http.NewRequest("POST", apiURL+"/CherwellAPI/api/V1/savebusinessobject", bytes.NewBuffer(jsonStr))
	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	freq.Header.Set("Content-Type", "application/json")
	freq.Header.Set("Authorization", "Bearer "+rt.AccessToken)

	fresp, err := client.Do(freq)
	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	fmt.Printf("Cherwell response: %v", fresp)
	defer fresp.Body.Close()

	fData, err := ioutil.ReadAll(fresp.Body)

	if err != nil {
		log.Printf("error: %v", err)
		return Response{
			Message: err.Error(),
		}, err
	}

	return Response{
		Message: string(fData),
	}, nil

}

func main() {
	lambda.Start(Handler)
}
