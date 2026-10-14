package util

import (
	"encoding/base64"
	"fmt"
	"time"

	sharedv1beta1 "github.com/datakit-dev/dtkt-sdk/sdk-go/proto/dtkt/shared/v1beta1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	PageFilterRequest interface {
		PageRequest
		FilterRequest
	}
	PageRequest interface {
		GetPageSize() int32
		GetPageToken() string
	}
	FilterRequest interface {
		GetFilter() string
	}
)

func NextPageTokenUUID(uid uuid.UUID) string {
	return base64.StdEncoding.EncodeToString(uid[:])
}

func ParsePageTokenUUID(pageToken string) (uid uuid.UUID, err error) {
	if pageToken == "" {
		return
	}

	b, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return
	}

	uid, err = uuid.FromBytes(b)
	if err != nil {
		return
	}

	if uid.Version() != 7 {
		err = fmt.Errorf("expected UUID version 7, got: %d", uid.Version())
		return
	}

	return
}

func ParsePageTokenRequestUUID(req PageRequest, defaultSize, minSize, maxSize int32) (uid uuid.UUID, s int32, err error) {
	uid, err = ParsePageTokenUUID(req.GetPageToken())
	if err != nil {
		return
	}

	s = GetRequestPageSize(req, defaultSize, minSize, maxSize)

	return
}

func NextPageToken(id int64, time time.Time) (string, error) {
	b, err := proto.Marshal(&sharedv1beta1.PageToken{
		Id:   id,
		Time: timestamppb.New(time),
	})
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func GetRequestPageSize(req PageRequest, defaultSize, minSize, maxSize int32) int32 {
	if req.GetPageSize() == 0 {
		return defaultSize
	}
	return max(min(req.GetPageSize(), maxSize), minSize)
}

func ParsePageTokenRequest(req PageRequest, defaultSize, minSize, maxSize int32) (id int64, time time.Time, size int32, err error) {
	size = GetRequestPageSize(req, defaultSize, minSize, maxSize)
	id, time, err = ParsePageToken(req.GetPageToken())
	return
}

func ParsePageToken(pageToken string) (id int64, time time.Time, err error) {
	if pageToken == "" {
		return
	}

	b, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return
	}

	pt := new(sharedv1beta1.PageToken)
	err = proto.Unmarshal(b, pt)
	if err != nil {
		return
	}

	id = pt.Id
	time = pt.Time.AsTime()

	return
}
