// Package salesforce
// This file has bulk related functionality that is internal to this package.
package salesforce

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/amp-labs/connectors/common"
)

const (
	Upsert BulkOperationMode = "upsert"
	Delete BulkOperationMode = "delete"

	JobStateAborted        = "Aborted"
	JobStateFailed         = "Failed"
	JobStateComplete       = "JobComplete"
	JobStateInProgress     = "InProgress"
	JobStateUploadComplete = "UploadComplete"

	sfIdFieldName    = "sf__Id"
	sfErrorFieldName = "sf__Error"
)

var (
	ErrKeyNotFound          = errors.New("key not found")
	ErrInvalidType          = errors.New("invalid type")
	ErrInvalidJobState      = errors.New("invalid job state")
	ErrUnsupportedMode      = errors.New("unsupported mode")
	ErrReadToByteFailed     = errors.New("failed to read data to bytes")
	ErrUnsupportedOperation = errors.New("unsupported operation")
)

// BulkOperationParams defines how we are writing data to a SaaS API.
type BulkOperationParams struct {
	// The name of the object we are writing, e.g. "Account"
	ObjectName string // required

	// The name of a field on the object which is an External ID. Provided in the case of upserts, not inserts
	ExternalIdField string // required

	// The path to the CSV file we are writing
	CSVData io.Reader // required

	// Salesforce operation mode
	Mode BulkOperationMode
}

type BulkOperationMode string

// BulkOperationResult is what's returned from writing data via the BulkOperation call.
type BulkOperationResult struct {
	// State is the state of the bulk job process
	State string `json:"state"`
	// JobId is the ID of the bulk job process
	JobId string `json:"jobId"`
}

type GetJobInfoResult struct {
	Id                     string            `json:"id"`
	Object                 string            `json:"object"`
	CreatedById            string            `json:"createdById"`
	ExternalIdFieldName    string            `json:"externalIdFieldName,omitempty"`
	State                  string            `json:"state"`
	Operation              BulkOperationMode `json:"operation"`
	ColumnDelimiter        string            `json:"columnDelimiter"`
	LineEnding             string            `json:"lineEnding"`
	NumberRecordsFailed    float64           `json:"numberRecordsFailed"`
	NumberRecordsProcessed float64           `json:"numberRecordsProcessed"`
	ErrorMessage           string            `json:"errorMessage"`

	ApexProcessingTime      float64 `json:"apexProcessingTime,omitempty"`
	ApiActiveProcessingTime float64 `json:"apiActiveProcessingTime,omitempty"`
	ApiVersion              float64 `json:"apiVersion,omitempty"`
	ConcurrencyMode         string  `json:"concurrencyMode,omitempty"`
	ContentType             string  `json:"contentType,omitempty"`
	CreatedDate             string  `json:"createdDate,omitempty"`
	JobType                 string  `json:"jobType,omitempty"`
	Retries                 float64 `json:"retries,omitempty"`
	SystemModstamp          string  `json:"systemModstamp,omitempty"`
	TotalProcessingTime     float64 `json:"totalProcessingTime,omitempty"`
	IsPkChunkingSupported   bool    `json:"isPkChunkingSupported,omitempty"`
}

func (r GetJobInfoResult) IsStatusDone() bool {
	return isStatusDone(r.State)
}

type FailInfo struct {
	FailureType   string              `json:"failureType"`
	FailedUpdates map[string][]string `json:"failedUpdates,omitempty"`
	FailedCreates map[string][]string `json:"failedCreates,omitempty"`
	Reason        string              `json:"reason,omitempty"`
}

type JobResults struct {
	JobId          string            `json:"jobId"`
	State          string            `json:"state"`
	FailureDetails *FailInfo         `json:"failureDetails,omitempty"`
	JobInfo        *GetJobInfoResult `json:"jobInfo,omitempty"`
	Message        string            `json:"message,omitempty"`
}

func (r JobResults) IsStatusDone() bool {
	return isStatusDone(r.State)
}

func isStatusDone(state string) bool {
	return state == JobStateComplete ||
		state == JobStateFailed ||
		state == JobStateAborted
}

// https://developer.salesforce.com/docs/atlas.en-us.api_asynch.meta/api_asynch/create_job.htm
func (c *Connector) createJob(ctx context.Context, body map[string]any) (*common.JSONHTTPResponse, error) {
	location, err := c.getRestApiURL("jobs/ingest")
	if err != nil {
		return nil, err
	}

	return c.Client.Post(ctx, location.String(), body)
}

// https://developer.salesforce.com/docs/atlas.en-us.api_asynch.meta/api_asynch/upload_job_data.htm
func (c *Connector) uploadCSV(ctx context.Context, jobId string, csvData io.Reader) ([]byte, error) {
	location, err := c.getRestApiURL(fmt.Sprintf("jobs/ingest/%s/batches", jobId))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(csvData)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to upload CSV to salesforce: %w",
			errors.Join(ErrReadToByteFailed, err),
		)
	}

	return c.Client.PutCSV(ctx, location.String(), data)
}

// https://developer.salesforce.com/docs/atlas.en-us.api_asynch.meta/api_asynch/close_job.htm
func (c *Connector) completeUpload(ctx context.Context, jobId string) (*common.JSONHTTPResponse, error) {
	location, err := c.getRestApiURL("jobs/ingest/" + jobId)
	if err != nil {
		return nil, err
	}

	return c.Client.Patch(ctx, location.String(), map[string]any{
		"state": JobStateUploadComplete,
	})
}

// https://developer.salesforce.com/docs/atlas.en-us.api_asynch.meta/api_asynch/get_job_failed_results.htm
func (c *Connector) getFailedJobResults(ctx context.Context, jobId string) (*http.Response, error) {
	location, err := c.getRestApiURL(fmt.Sprintf("jobs/ingest/%s/failedResults", jobId))
	if err != nil {
		return nil, err
	}

	req, err := common.MakeJSONGetRequest(ctx, location.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get request: %w", err)
	}

	// Get the connector's JSONHTTPClient, which is a special HTTPClient that handles JSON responses,
	// and use it's underlying http.Client to make the request.
	return c.Client.HTTPClient.Client.Do(req)
}

//nolint:funlen,cyclop
func (c *Connector) getPartialFailureDetails(ctx context.Context, jobInfo *GetJobInfoResult) (*JobResults, error) {
	// Query Salesforce to get partial failure details
	res, err := c.getFailedJobResults(ctx, jobInfo.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job results: %w", err)
	}
	defer res.Body.Close()
	// if partially failed, we need to get the error message for each record
	reader := csv.NewReader(res.Body)

	failInfo := &FailInfo{
		FailureType:   "Partial",
		FailedUpdates: make(map[string][]string),
		FailedCreates: make(map[string][]string),
	}

	var sfIdColIdx, sfErrorColIdx, externalIdColIdx int

	rowIdx := 0

	for {
		record, err := reader.Read()
		if err != nil {
			// end of file
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, err
		}

		fieldNames := []string{sfIdFieldName, sfErrorFieldName}

		if jobInfo.ExternalIdFieldName != "" {
			fieldNames = append(fieldNames, jobInfo.ExternalIdFieldName)
		}

		// Get column index of sf__Id, sf__Error, and externalIdFieldName in header row
		// Salesforce API responses may not be consistent with the order of columns, so we need to get the index
		if rowIdx == 0 {
			indiceMap, err := getColumnIndice(record, fieldNames)
			if err != nil {
				return nil, err
			}

			sfIdColIdx = indiceMap[sfIdFieldName]
			sfErrorColIdx = indiceMap[sfErrorFieldName]

			if jobInfo.ExternalIdFieldName != "" {
				externalIdColIdx = indiceMap[jobInfo.ExternalIdFieldName]
			}

			rowIdx++

			continue
		}

		sfId := record[sfIdColIdx]
		failureMap := failInfo.FailedUpdates
		errMsg := record[sfErrorColIdx]

		if sfId == "" {
			// If sf__Id is empty, it means the record is not updated, so it's a create failure
			failureMap = failInfo.FailedCreates
		}

		var referenceId string

		switch jobInfo.Operation {
		case Upsert:
			// for bulkwrite, we will have ExternalIdFieldName
			referenceId = record[externalIdColIdx]
		case Delete:
			// for bulkdelete, we will have sf__Id as reference
			referenceId = sfId
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedOperation, jobInfo.Operation)
		}

		failureMap[errMsg] = append(failureMap[errMsg], referenceId)
	}

	return &JobResults{
		FailureDetails: failInfo,
		JobId:          jobInfo.Id,
		State:          jobInfo.State,
		JobInfo:        jobInfo,
		Message:        "Some records are not processed successfully. Please refer to the 'failureDetails' for more details.",
	}, nil
}

func getColumnIndice(record []string, columnNames []string) (map[string]int, error) {
	indices := make(map[string]int)

	for _, columnName := range columnNames {
		found := false

		for j, value := range record {
			if strings.EqualFold(value, columnName) {
				indices[columnName] = j
				found = true

				break
			}
		}

		if !found {
			return nil, fmt.Errorf("%w: '%s'", ErrKeyNotFound, columnName)
		}
	}

	return indices, nil
}

func getIncompleteJobMessage(jobInfo *GetJobInfoResult) string {
	switch {
	case jobInfo.State == JobStateInProgress || jobInfo.State == JobStateUploadComplete:
		return "Job is still in progress. Please try again later."
	case jobInfo.State == JobStateAborted:
		return "Job aborted. Please refer to the JobInfo for more details."
	case jobInfo.State == JobStateFailed:
		//nolint:lll
		return "No records processed successfully. This is likely due the CSV being empty or issues with CSV column names."
	default:
		return "Job is in an unknown state."
	}
}

//nolint:funlen,cyclop
func (c *Connector) bulkOperation(
	ctx context.Context,
	params BulkOperationParams,
	jobBody map[string]any,
) (*BulkOperationResult, error) {
	res, err := c.createJob(ctx, jobBody)
	if err != nil {
		return nil, fmt.Errorf("createJob failed: %w", err)
	}

	body, ok := res.Body() // nolint:varnamelen
	if !ok {
		return nil, fmt.Errorf("createJob failed: %w", common.ErrEmptyJSONHTTPResponse)
	}

	resObject, err := body.GetObject()
	if err != nil {
		return nil, fmt.Errorf(
			"parsing result of createJob failed: %w",
			errors.Join(err, common.ErrParseError),
		)
	}

	state, err := resObject["state"].GetString()
	if err != nil {
		unpacked, _ := resObject["state"].Unpack()

		return nil, fmt.Errorf(
			"%w: expected salesforce job state to be string in response, got %T",
			ErrInvalidType,
			unpacked,
		)
	}

	if strings.ToLower(state) != "open" {
		return nil, fmt.Errorf("%w: expected job state to be open, got %s", ErrInvalidJobState, state)
	}

	jobId, err := resObject["id"].GetString()
	if err != nil {
		unpacked, _ := resObject["id"].Unpack()

		return nil, fmt.Errorf(
			"%w: expected salesforce job id to be string in response, got %T",
			ErrInvalidType,
			unpacked)
	}

	// upload csv and there is no response body other than status code
	_, err = c.uploadCSV(ctx, jobId, params.CSVData)
	if err != nil {
		return nil, fmt.Errorf("uploadCSV failed: %w", err)
	}

	data, err := c.completeUpload(ctx, jobId)
	if err != nil {
		return nil, fmt.Errorf("completeUpload failed: %w", err)
	}

	body, ok = data.Body()
	if !ok {
		return nil, fmt.Errorf("completeUpload failed: %w", common.ErrEmptyJSONHTTPResponse)
	}

	dataObject, err := body.GetObject()
	if err != nil {
		return nil, fmt.Errorf("parsing result of completeUpload failed: %w", errors.Join(err, common.ErrParseError))
	}

	updatedJobId, err := dataObject["id"].GetString()
	if err != nil {
		unpacked, _ := dataObject["id"].Unpack()

		return nil, fmt.Errorf(
			"%w: expected salesforce job id to be string in response, got %T",
			ErrInvalidType,
			unpacked,
		)
	}

	completeState, err := dataObject["state"].GetString() //nolint:varnamelen
	if err != nil {
		unpacked, _ := resObject["state"].Unpack()

		return nil, fmt.Errorf(
			"%w: expected salesforce job state to be string in response, got %T",
			ErrInvalidType,
			unpacked,
		)
	}

	return &BulkOperationResult{
		JobId: updatedJobId,
		State: completeState,
	}, nil
}
