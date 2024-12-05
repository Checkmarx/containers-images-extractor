package imagesExtractor

import (
	"archive/tar"
	"compress/gzip"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
)

const DirToExtractTar = "extracted_tar"

func extractTar(tarPath string) (string, error) {
	r, err := os.Open(tarPath)
	if err != nil {
		log.Err(err).Msgf("Could not extract Tar `%s`", tarPath)
	}

	extractDir := filepath.Join(filepath.Dir(tarPath), DirToExtractTar)
	err = os.MkdirAll(extractDir, os.ModePerm)
	if err != nil {
		log.Err(err).Msgf("Could not create directory `%s`", extractDir)
		return "", err
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		log.Err(err).Msgf("Could not create gzip reader `%s`.", extractDir)
		return "", err
	}
	defer func(gzr *gzip.Reader) {
		err = gzr.Close()
		if err != nil {
			log.Warn().Msgf("error whole closing gzip reader, err: %+v", err)
		}
	}(gzr)

	tr := tar.NewReader(gzr)

	for {
		header, headerErr := tr.Next()
		switch {

		case headerErr == io.EOF:
			return extractDir, nil

		case headerErr != nil:
			log.Err(headerErr).Msg("could not get next header.")
			return "", headerErr

		case header == nil:
			continue
		}

		target := filepath.Join(extractDir, header.Name)

		switch header.Typeflag {

		case tar.TypeDir:
			if _, dirFileErr := os.Stat(target); dirFileErr != nil {
				if dirFileErr = os.MkdirAll(target, 0755); dirFileErr != nil {
					log.Err(err).Msgf("could not create dir: %s", target)
					return "", dirFileErr
				}
			}

		case tar.TypeReg:
			f, regFileErr := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if regFileErr != nil {
				log.Err(regFileErr).Msgf("could not open file: %s", target)
				return "", regFileErr
			}

			if _, regFileErr = io.Copy(f, tr); regFileErr != nil {
				log.Err(regFileErr).Msgf("could not copy file: %s", f.Name())
				return "", regFileErr
			}

			regFileErr = f.Close()
			if regFileErr != nil {
				log.Warn().Msgf("error whole closing file %s, err: %+v", f.Name(), err)
				return "", regFileErr
			}
		}
	}
}
