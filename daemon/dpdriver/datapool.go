package dpdriver

import (
	"errors"
	"fmt"
	"github.com/asiainfoLDP/datahub/cmd"
	"github.com/asiainfoLDP/datahub/ds"
	log "github.com/asiainfoLDP/datahub/utils/clog"
	"os"
	"reflect"
)

type DatapoolDriver interface {
	GetDestFileName(dpconn, itemlocation, filename string) (destfilename, tmpdir, tmpfile string)
	StoreFile(status, filename, dpconn, dp, itemlocation, destfile string) string
	GetFileTobeSend(dpconn, dpname, itemlocation, tagdetail string) (filepathname string)
	CheckItemLocation(datapool, dpconn, itemdesc string) error
	CheckDataAndGetSize(dpconn, itemlocation, fileName string) (exist bool, size int64, err error)
	GetDpOtherData(allotherdata *[]ds.DpOtherData, itemslocation map[string]string, dpconn string) (err error)
	CheckDpConnect(dpconn, connstr string) (normal bool, err error)
}

type Datapool struct {
	driver DatapoolDriver
}

var datapooldrivers = make(map[string]DatapoolDriver)

const gDpPath string = cmd.GstrDpPath

func register(name string, datapooldriver DatapoolDriver) {
	if datapooldriver == nil {
		panic("dpdriver: Register datapooldriver is nil")
	}
	if _, dup := datapooldrivers[name]; dup {
		panic("dpdriver: Register called twice for datapooldriver " + name)
	}
	datapooldrivers[name] = datapooldriver
}

func New(name string) (*Datapool, error) {
	datapooldriver, ok := datapooldrivers[name]
	for k, _ := range datapooldrivers {
		log.Debug(k, datapooldrivers[k], reflect.TypeOf(datapooldrivers[k]))
	}
	if !ok {
		s := fmt.Sprintf("Can't find datapooldriver %v", name)
		log.Error(s)
		return nil, errors.New(s)
	}
	return &Datapool{driver: datapooldriver}, nil
}

func (datapool *Datapool) GetDestFileName(dpconn, itemlocation, filename string) (destfilename, tmpdir, tmpfile string) {
	return datapool.driver.GetDestFileName(dpconn, itemlocation, filename)
}

func (datapool *Datapool) StoreFile(status, filename, dpconn, dp, itemlocation, destfile string) string {
	return datapool.driver.StoreFile(status, filename, dpconn, dp, itemlocation, destfile)
}

func (datapool *Datapool) GetFileTobeSend(dpconn, dpname, itemlocation, tagdetail string) (filepathname string) {
	return datapool.driver.GetFileTobeSend(dpconn, dpname, itemlocation, tagdetail)
}

func (datapool *Datapool) CheckItemLocation(datapoolname, dpconn, itemlocation string) error {
	return datapool.driver.CheckItemLocation(datapoolname, dpconn, itemlocation)
}

func (datapool *Datapool) CheckDataAndGetSize(dpconn, itemlocation, fileName string) (exist bool, size int64, err error) {
	return datapool.driver.CheckDataAndGetSize(dpconn, itemlocation, fileName)
}

func (datapool *Datapool) GetDpOtherData(allotherdata *[]ds.DpOtherData, itemslocation map[string]string, dpconn string) (err error) {
	return datapool.driver.GetDpOtherData(allotherdata, itemslocation, dpconn)
}

func (datapool *Datapool) CheckDpConnect(dpconn, connstr string) (normal bool, err error) {
	return datapool.driver.CheckDpConnect(dpconn, connstr)
}

/*func (handler *Handler) DoUnbind(myServiceInfo *ServiceInfo, mycredentials *Credentials) error {
	return handler.driver.DoUnbind(myServiceInfo, mycredentials)
}
*/

func Env(name string, required bool) string {
	s := os.Getenv(name)
	if required && s == "" {
		panic("env variable required, " + name)
	}
	log.Debugf("[env][%s] %s\n", name, s)
	return s
}

func SetEnv(name string, value string) string {
	if e := os.Setenv(name, value); e != nil {
		log.Errorf("[setenv][%s] %s, error:%v\n", name, value, e)
		return ""
	}

	log.Debugf("[setenv][%s] %s\n", name, value)
	return name
}
