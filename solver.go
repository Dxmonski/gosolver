package gosolver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var Client = &http.Client{}
var Service string

func SolveHCaptcha(data SolverData) Result {
	url := ""
	task := ""

	Service = strings.ToLower(data.Service)

	switch Service {
	case "capsolver":
		url = CapsolverBaseURL + "/createTask"
	case "capmonster":
		url = CapmonsterBaseURL + "/createTask" // gonna finish this implementation soon aswell as 2capthca.
	}

	if data.Proxy == "" {
		task = "HCaptchaTaskProxyLess"
	} else {
		task = "HCaptchaTurboTask"
	}

	payload := TaskPayload{
		ClientKey: data.ClientKey,
		Task: HCaptchaTask{
			Type:        task,
			WebsiteURL:  data.WebURL,
			WebsiteKey:  data.SiteKey,
			IsInvisible: true,
			EnterprisePayload: struct {
				Rqdata string `json:"rqdata"`
			}{Rqdata: ""},
			Proxy:     "",
			UserAgent: "",
		},
	}

	if data.SiteKey == "" {
		fmt.Printf("[GoSolver] - a website key is required with the context.\n")
		return Result{}
	}

	if data.UA != "" {
		payload.Task.UserAgent = data.UA
	}

	if data.Proxy != "" {
		payload.Task.Proxy = data.Proxy
	}

	if data.RQData != "" {
		payload.Task.EnterprisePayload.Rqdata = data.RQData
	}

	p, err := json.Marshal(payload)

	if err != nil {
		fmt.Printf("[GoSolver] - Error while marshaling payload: %s\n", err)
		return Result{}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(p))

	if err != nil {
		fmt.Printf("[GoSolver] - Failed to create new HTTP request: %s\n", err)
		return Result{}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := Client.Do(req)

	if err != nil {
		fmt.Printf("[GoSolver] - Failed to perform client request: %s\n", err)
		return Result{}
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("[GoSolver] - Failed to read response body: %s\n", err)
		return Result{}
	}

	var response Created

	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("[GoSolver] - Failed to unmarshal event: %s\n", err)
		return Result{}
	}

	if response.ErrorID != 0 {
		if err := ProcessError(body); err != nil {
			fmt.Printf("[GoSolver] - Unknown error: %s\n", err)
			return Result{}
		}
	}

	if response.TaskID != "" && response.ErrorID == 0 {
		for {
			res, err := ProcessTask(TaskResult{
				Key:    data.ClientKey,
				TaskID: response.TaskID,
			})

			switch res.CapStatus {
			case "ready":
				fmt.Printf("[GoSolver] - Successfully solved captcha: %s\n", res.CapResp[:20]+"...")
				return Result{
					Captchakey:      res.CapKey,
					CaptchaResponse: res.CapResp,
				}
			case "failed":
				fmt.Printf("[GoSolver] - Failed to solve captcha: %s\n", err)
				return Result{}
			}

			time.Sleep(3 * time.Second)
		}
	}

	return Result{}
}

func ProcessTask(rq TaskResult) (Context, error) {
	url := ""

	switch Service {
	case "capsolver":
		url = CapsolverBaseURL + "/getTaskResult"
	case "capmonster":
		url = CapmonsterBaseURL + "/getTaskResult "
	}

	payload, err := json.Marshal(rq)

	if err != nil {
		return Context{}, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))

	if err != nil {
		return Context{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := Client.Do(req)

	if err != nil {
		return Context{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return Context{}, err
	}

	var r Finished

	if err := json.Unmarshal(body, &r); err != nil {
		return Context{}, err
	}

	return Context{
		CapKey:    r.Solution.CaptchaKey,
		CapResp:   r.Solution.CaptchaResp,
		CapStatus: r.Status,
	}, nil
}

func ProcessError(resp []byte) error {
	var err Error
	var errorMessages map[string]string

	if err := json.Unmarshal(resp, &err); err != nil {
		fmt.Printf("Failed to unmarshal event: %s\n", err)
		return err
	}

	switch Service {
	case "capsolver":
		errorMessages = map[string]string{
			"ERROR_SERVICE_UNAVAILABLE": "Service is currently unavailable",
			"ERROR_RATE_LIMIT":          "Request frequency/rate limit exceeded",
			"ERROR_INVALID_TASK_DATA":   "Data parsing failure",
			"ERROR_BAD_REQUEST":         "Request error, please contact customer service if it persists",
			"ERROR_TASKID_INVALID":      "Task ID does not exist or is invalid",
			"ERROR_TASK_TIMEOUT":        "The task has timed out",
			"ERROR_SETTLEMENT_FAILED":   "Mission point settlement failure",
			"ERROR_KEY_DENIED_ACCESS":   "Wrong account key, please check your clientKey",
			"ERROR_ZERO_BALANCE":        "Insufficient account balance, please top up",
			"ERROR_TASK_NOT_SUPPORTED":  "Task types are not supported",
			"ERROR_CAPTCHA_UNSOLVABLE":  "Captcha not recognized, please try again",
			"ERROR_UNKNOWN_QUESTION":    "Wrong Question ID, the task cannot be processed",
			"ERROR_PROXY_BANNED":        "Proxy IP banned by target service",
			"ERROR_INVALID_IMAGE":       "Image size does not meet the requirements",
			"ERROR_PARSE_IMAGE_FAIL":    "Failed to parse image, please check the image BASE64 encoding",
			"ERROR_IP_BANNED":           "Your IP has been blocked for 10 minutes",
			"ERROR_KEY_TEMP_BLOCKED":    "Due to too many error requests, you have been blocked temporarily",
		}
	case "capmonster":
		errorMessages = map[string]string{
			"INVALID KEY":               "Account authorization key not found or has an incorrect format",
			"ZERO IMAGE SIZE":           "The size of the captcha you are uploading is less than 100 bytes",
			"BIG IMAGE SIZE":            "The size of the captcha you are uploading is more than 500,000 bytes",
			"NO FUNDS":                  "Account has zero balance. Add funds to continue recognition",
			"REQUEST IS NOT ALLOWED":    "Request with the current account key is not allowed from your IP. Open your account settings and add your IP to the trusted list",
			"CAPTCHA UNSOLVABLE":        "This type of captchas is not supported by the service or the image does not contain an answer. It could also mean that the image is corrupted or was incorrectly rendered",
			"CAPTCHA ID IS NOT FOUND":   "The captcha that you are requesting was not found. Make sure you are requesting a status update only within 5 minutes of uploading",
			"CAPTCHA IS NOT READY":      "The captcha has not yet been solved",
			"IP BANNED":                 "You have exceeded the limit of requests with the wrong api key, check the correctness of your api key in the control panel and after some time, try again",
			"INCORRECT METHOD":          "Incorrect captcha type (value of the «type» parameter)",
			"REQUEST LIMIT EXCEEDED":    "You have exceeded the limit of requests to receive an answer for one task. Try to request the result of the task no more than 1 time in 2 seconds",
			"THE DOMAIN IS NOT ALLOWED": "Captcha from some domains cannot be solved in CapMonster Cloud. If you try to create a task for such a domain, this error will return",
			"THE TOKEN IS EXPIRED":      "Captcha provider server reported that the additional token has expired. Try creating a task with a new token",
			"NO FREE SERVERS":           "At the moment there are no available servers for recognizing this task. Try again after a while",
			"INVALID RECAPTCHA SITEKEY": "Invalid sitekey",
			"INVALID RECAPTCHA DOMAIN":  "Invalid domain for sitekey",
			"RECAPTCHA TIMEOUT":         "The timeout of the ReCaptcha recognition has been exceeded, most likely due to a slow proxy or Google server",
			"YOUR IP IS BLOCKED":        "Your IP is not allowed to access this API due to a large number of errors",
			"FAILED TO CONNECT PROXY":   "Unable to connect to the proxy server, connection timeout",
			"THE PROXY IP IS BANNED":    "The proxy IP is banned in the target captcha service",
			"INCORRECT TASK TYPE":       "The task type is incorrect or not supported. Check the «type» property in the task object",
			"ERROR_TASK_ABSENT":         "Task object not found or invalid JSON was sent in createTask request",
		}
	default:
		errorMessages = map[string]string{
			"UNKNOWN ERROR": "Unknown error occurred",
		}
	}

	errorMessage, found := errorMessages[err.ErrorCode]
	if !found {
		errorMessage = "Unknown error occurred"
	}

	fmt.Printf("[GoSolver] - Failed to solve captcha: %s\n", errorMessage)

	return nil
}
