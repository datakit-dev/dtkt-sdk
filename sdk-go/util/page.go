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

type PageTokenRequest interface {
	GetPageSize() int32
	GetPageToken() string
}

func NextPageTokenUUIDV7(uid uuid.UUID) string {
	return base64.StdEncoding.EncodeToString(uid[:])
}

func ParsePageTokenUUIDV7(pageToken string) (uid uuid.UUID, err error) {
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
		err = fmt.Errorf("invalid UUID version: %d", uid.Version())
		return
	}

	return
}

func ParsePageTokenRequestUUIDV7[T PageTokenRequest](req T, defaultSize, minSize, maxSize int32) (uid uuid.UUID, s int32, err error) {
	uid, err = ParsePageTokenUUIDV7(req.GetPageToken())
	if err != nil {
		return
	}

	s = GetPageSizeRequest(req, defaultSize, minSize, maxSize)

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

func GetPageSizeRequest[T PageTokenRequest](req T, defaultSize, minSize, maxSize int32) int32 {
	if req.GetPageSize() == 0 {
		return defaultSize
	}
	return max(min(req.GetPageSize(), maxSize), minSize)
}

func ParsePageTokenRequest[T PageTokenRequest](req T, defaultSize, minSize, maxSize int32) (i int64, t time.Time, s int32, err error) {
	s = GetPageSizeRequest(req, defaultSize, minSize, maxSize)

	i, t, err = ParsePageToken(req.GetPageToken())
	if err != nil {
		return
	}
	return
}

func ParsePageToken(pageToken string) (i int64, t time.Time, err error) {
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
	i = pt.Id
	t = pt.Time.AsTime()
	return
}
