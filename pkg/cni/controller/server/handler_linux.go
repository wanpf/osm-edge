package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/containernetworking/cni/pkg/skel"
	log "github.com/sirupsen/logrus"
)

func (s *server) PodCreated(w http.ResponseWriter, req *http.Request) {
	bs, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	args := skel.CmdArgs{}
	err = json.Unmarshal(bs, &args)
	log.Infof("cni called create with args: %+v", args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	err = s.CmdAdd(&args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) PodDeleted(w http.ResponseWriter, req *http.Request) {
	bs, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	args := skel.CmdArgs{}
	err = json.Unmarshal(bs, &args)
	log.Infof("cni called delete with args: %+v", args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	err = s.CmdDelete(&args)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *server) TransferFd(w http.ResponseWriter, req *http.Request) {
	_, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
	}
	s.transferFds()
	w.WriteHeader(200)
	_, _ = w.Write([]byte("ok"))
}
