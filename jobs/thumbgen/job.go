package thumbgen

import (
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"log"

	jpgImage "image/jpeg"

	"github.com/disintegration/imageorient"
	"github.com/disintegration/imaging"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/uploadexpress/app/config"
	"github.com/uploadexpress/app/helpers/params"
	"github.com/uploadexpress/app/models"
	"github.com/uploadexpress/app/services/s3"
	"github.com/uploadexpress/app/store"
)

var SupportedThumbnailTypes = matchers.Map{
	matchers.TypeJpeg: matchers.Jpeg,
	matchers.TypePng:  matchers.Png,
}

type ThumbnailGenerator struct {
	params params.M
}

func NewThumbnailGenerator(params params.M) *ThumbnailGenerator {
	return &ThumbnailGenerator{
		params,
	}
}

func (tg ThumbnailGenerator) Name() string {
	return "thumbnail_generator"
}

func (tg ThumbnailGenerator) Execute(store store.Store, configuration *viper.Viper) {
	file := tg.params["file"].(models.File)
	uploadId := tg.params["uploadId"].(string)

	awsConfig := config.NewAwsConfigurationFromConfig(configuration)

	reader, err := s3.GetObjectHeader(awsConfig, uploadId, file)
	if err != nil {
		logrus.Error("could not fetch the header for file" + file.Id)
	}

	fileHeader, err := ioutil.ReadAll(reader)
	if err != nil {
		logrus.Error("could not read the header for file" + file.Id)
	}

	if filetype.MatchesMap(fileHeader, SupportedThumbnailTypes) {
		s3image, err := s3.GetObjectReader(awsConfig, uploadId, file)
		if err != nil {
			logrus.Error("could not fetch the image " + file.Name)
			return
		}

		var img image.Image
		img, _, err = imageorient.Decode(s3image)
		if err != nil {
			log.Fatalf("imageorient.Decode failed: %v", err)
			return
		}

		var dstImage256 *image.NRGBA
		var dstImage1024 *image.NRGBA
		dstImage256 = resizeImage(img, 256)
		dstImage1024 = resizeImage(img, 1024)

		url256, err := encodeImage(awsConfig, uploadId, file, dstImage256, 100)
		if err != nil {
			fmt.Println("failed to upload buffer", err)
			return
		}

		url1024, err := encodeImage(awsConfig, uploadId, file, dstImage1024, 60)
		if err != nil {
			fmt.Println("failed to upload buffer", err)
			return
		}

		err = store.AttachPreview(uploadId, file.Id, url1024, url256, dstImage256.Rect.Max.X, dstImage256.Rect.Max.Y)
		if err != nil {
			logrus.Error(err.Error())
			return
		}
	}
}

func encodeImage(awsConfig config.AwsConfiguration, uploadId string, file models.File, nrgba *image.NRGBA, quality int) (string, error) {
	buff := new(bytes.Buffer)
	err := jpgImage.Encode(buff, nrgba, &jpgImage.Options{
		Quality: quality,
	})
	if err != nil {
		return "", err
	}
	reader := bytes.NewReader(buff.Bytes())

	url, err := s3.PutPublicObject(awsConfig, fmt.Sprintf("previews/%s/%s_%d.png", uploadId, file.Id, nrgba.Rect.Max.X), ioutil.NopCloser(reader))
	if err != nil {
		logrus.Error(err.Error())
	}

	return url, nil
}

func resizeImage(img image.Image, size int) *image.NRGBA {
	width := img.Bounds().Max.X
	height := img.Bounds().Max.Y

	var resizedImage *image.NRGBA
	if width > height {
		resizedImage = imaging.Resize(img, size, 0, imaging.Lanczos)
	} else {
		resizedImage = imaging.Resize(img, 0, size, imaging.Lanczos)
	}

	return resizedImage
}