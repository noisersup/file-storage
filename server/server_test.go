package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/noisersup/encryptedfs-api/models"
	"github.com/stretchr/testify/assert"
)

func Test_GetFile(t *testing.T) {
	mockDB := MockDB{}
	s := Server{maxUpload: 1024 << 20, db: &mockDB, filesPath: "../testdata/files"}

	expected := []byte("foo.txt content")
	req := httptest.NewRequest(http.MethodGet, "/drive/test/foo.txt", nil)
	w := httptest.NewRecorder()

	s.GetFile(w, req, []string{"test/foo.txt"}, "user1")

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	data = data[:len(data)-1] //remove LF byte from response
	assert.NoError(t, err)

	assert.Equal(t, data, expected)
}

type MockDB struct {
}

func (m *MockDB) Close() {
}

func (m *MockDB) NewFile(pathNames []string, key []byte, duplicate int, isDirectory bool, userRoot uuid.UUID) error {
	return nil
}

func (m *MockDB) GetFile(pathNames []string, userRoot uuid.UUID) (*models.File, error) {
	if len(pathNames) == 0 {
		return nil, errors.New("pathNames empty")
	}

	user1ID := uuid.MustParse("0bb34349-a3f7-4221-ba6e-3dcd3ca78f30")
	user2ID := uuid.MustParse("01fb863a-ceeb-4b28-89b4-7dfe75d72961")

	fTest := models.File{
		Id:          uuid.MustParse("293fe451-b313-4c51-9fad-7d51e602af9b"),
		Name:        "test",
		ParentId:    user1ID,
		Duplicate:   0,
		IsDirectory: true,
	}

	fFoo := models.File{
		Id:          uuid.MustParse("78fbfd78-2511-4c93-a54f-68f22bc350c5"),
		Name:        "foo.txt",
		Hash:        "8436ae2cf3450e057fbe3a370528edbfe593dd702bdeb8cc0636a697eeee8b71",
		ParentId:    uuid.MustParse("293fe451-b313-4c51-9fad-7d51e602af9b"),
		Duplicate:   0,
		IsDirectory: false,
	}

	fBar := models.File{
		Id:          uuid.MustParse("1feac6c9-f1d8-4c1c-ac0e-f7c2f0a0b96b"),
		Name:        "bar.md",
		Hash:        "dfa1ee40529a430d1884af01f268d9784e1defab18bc4b6868b4f6640500c8b6",
		ParentId:    user2ID,
		Duplicate:   0,
		IsDirectory: false,
	}

	switch userRoot {
	case user1ID:
		if arraysEqual(pathNames, []string{"test", "foo.txt"}) {
			return &fFoo, nil
		}
		if arraysEqual(pathNames, []string{"test"}) {
			return &fTest, nil
		}
	case user2ID:
		if arraysEqual(pathNames, []string{"bar.md"}) {
			return &fBar, nil
		}
	}

	return nil, errors.New("File not found")

}

func arraysEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (m *MockDB) ListDirectory(id ...uuid.UUID) ([]models.File, error) {

	return nil, nil
}

func (m *MockDB) DeleteFile(pathNames []string, userRoot uuid.UUID) error {
	return nil
}

func (m *MockDB) NewUser(username, hashedPassword string) error {
	return nil

}

func (m *MockDB) GetPasswordOfUser(username string) (string, error) {
	return "", nil

}

func (m *MockDB) GetKey(username string) ([]byte, error) {
	var b64Decoded string

	switch username {
	case "user1":
		b64Decoded = "diZQ2zWrCyAT4B2aLAB0k+SMllBnz0xYJa/25wQxl3U="
	case "user2":
		b64Decoded = "TMBRuOMn3YjlNBxZ/zqxY5amg1da4RL9I+YZNh1ryoI="
	default:
		return nil, fmt.Errorf("user %s not found", username)
	}

	key, err := base64.StdEncoding.DecodeString(b64Decoded)
	if err != nil {
		return nil, err
	}
	return key, err
}

func (m *MockDB) GetRoot(username string) (uuid.UUID, error) {
	switch username {
	case "user1":
		return uuid.MustParse("0bb34349-a3f7-4221-ba6e-3dcd3ca78f30"), nil
	case "user2":
		return uuid.MustParse("01fb863a-ceeb-4b28-89b4-7dfe75d72961"), nil
	}
	return uuid.UUID{}, fmt.Errorf("user %s not found", username)
}
