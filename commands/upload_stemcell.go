package commands

import (
	"fmt"
	"path/filepath"

	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/formcontent"
	"github.com/pivotal-cf/om/network"
	"github.com/pivotal-cf/om/validator"

	"strconv"
)

const maxStemcellUploadRetries = 2

type UploadStemcell struct {
	multipart multipart
	logger    logger
	service   uploadStemcellService
	Options   struct {
		Stemcell string `long:"stemcell" short:"s" required:"true" description:"path to stemcell"`
		Force    bool   `long:"force"    short:"f"                 description:"upload stemcell even if it already exists on the target Ops Manager"`
		Floating bool   `long:"floating"                           description:"assigns the stemcell to all compatible products "`
		Shasum   string `long:"shasum" description:"shasum of the provided stemcell file to be used for validation"`
	}
}

//go:generate counterfeiter -o ./fakes/multipart.go --fake-name Multipart . multipart
type multipart interface {
	Finalize() formcontent.ContentSubmission
	Reset()
	AddFile(key, path string) error
	AddField(key, value string) error
}

//go:generate counterfeiter -o ./fakes/upload_stemcell_service.go --fake-name UploadStemcellService . uploadStemcellService
type uploadStemcellService interface {
	UploadStemcell(api.StemcellUploadInput) (api.StemcellUploadOutput, error)
	GetDiagnosticReport() (api.DiagnosticReport, error)
}

func NewUploadStemcell(multipart multipart, service uploadStemcellService, logger logger) UploadStemcell {
	return UploadStemcell{
		multipart: multipart,
		logger:    logger,
		service:   service,
	}
}

func (us UploadStemcell) Usage() jhanda.Usage {
	return jhanda.Usage{
		Description:      "This command will upload a stemcell to the target Ops Manager. Unless the force flag is used, if the stemcell already exists that upload will be skipped",
		ShortDescription: "uploads a given stemcell to the Ops Manager targeted",
		Flags:            us.Options,
	}
}

func (us UploadStemcell) Execute(args []string) error {
	us.Options.Floating = true

	if _, err := ParseArgs(&us.Options, args); err != nil {
		return fmt.Errorf("could not parse upload-stemcell flags: %s", err)
	}

	if us.Options.Shasum != "" {
		shaValidator := validator.NewSHA256Calculator()
		shasum, err := shaValidator.Checksum(us.Options.Stemcell)

		if err != nil {
			return err
		}

		if shasum != us.Options.Shasum {
			return fmt.Errorf("expected shasum %s does not match file shasum %s", us.Options.Shasum, shasum)
		}

		us.logger.Printf("expected shasum matches stemcell shasum.")
	}

	if !us.Options.Force {
		us.logger.Printf("processing stemcell")
		report, err := us.service.GetDiagnosticReport()
		if err != nil {
			switch err.(type) {
			case api.DiagnosticReportUnavailable:
				us.logger.Printf("%s", err)
			default:
				return fmt.Errorf("failed to get diagnostic report: %s", err)
			}
		}

		for _, stemcell := range report.Stemcells {
			if stemcell == filepath.Base(us.Options.Stemcell) {
				us.logger.Printf("stemcell has already been uploaded")
				return nil
			}
		}
	}

	var err error
	for i := 0; i <= maxStemcellUploadRetries; i++ {
		err = us.multipart.AddFile("stemcell[file]", us.Options.Stemcell)
		if err != nil {
			return fmt.Errorf("failed to load stemcell: %s", err)
		}

		err = us.multipart.AddField("stemcell[floating]", strconv.FormatBool(us.Options.Floating))
		if err != nil {
			return fmt.Errorf("failed to load stemcell: %s", err)
		}

		submission := us.multipart.Finalize()
		if err != nil {
			return fmt.Errorf("failed to create multipart form: %s", err)
		}

		us.logger.Printf("beginning stemcell upload to Ops Manager")

		_, err = us.service.UploadStemcell(api.StemcellUploadInput{
			Stemcell:      submission.Content,
			ContentType:   submission.ContentType,
			ContentLength: submission.ContentLength,
		})
		if network.CanRetry(err) && i < maxStemcellUploadRetries {
			us.logger.Printf("retrying stemcell upload after error: %s\n", err)
			us.multipart.Reset()
		} else {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to upload stemcell: %s", err)
	}

	us.logger.Printf("finished upload")

	return nil
}
