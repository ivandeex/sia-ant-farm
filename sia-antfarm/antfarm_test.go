package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/NebulousLabs/Sia-Ant-Farm/ant"
	"gitlab.com/NebulousLabs/Sia-Ant-Farm/test"
	"gitlab.com/NebulousLabs/Sia/modules"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
)

// renterAntName defines name of the renter ant
const renterAntName = "Renter"

// createBasicRenterAntfarmConfig creates default basic antfarm config for
// running renter tests
func createBasicRenterAntfarmConfig(dataDir string) AntfarmConfig {
	antFarmAddr := test.RandomLocalAddress()
	antFarmDir := filepath.Join(dataDir, "antfarm-data")
	antDirs := test.AntDirs(dataDir, 7)
	config := AntfarmConfig{
		ListenAddress: antFarmAddr,
		DataDir:       antFarmDir,
		AntConfigs: []ant.AntConfig{
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[0],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs: []string{"gateway", "miner"},
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[1],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs:            []string{"host"},
				DesiredCurrency: 100000,
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[2],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs:            []string{"host"},
				DesiredCurrency: 100000,
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[3],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs:            []string{"host"},
				DesiredCurrency: 100000,
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[4],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs:            []string{"host"},
				DesiredCurrency: 100000,
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[5],
					RPCAddr:                  test.RandomLocalAddress(),
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs:            []string{"host"},
				DesiredCurrency: 100000,
			},
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress:      true,
					RenterDisableIPViolationCheck: true,
					DataDir:                       antDirs[6],
					RPCAddr:                       test.RandomLocalAddress(),
					SiadPath:                      test.TestSiadFilename,
				},
				Jobs:            []string{"renter"},
				DesiredCurrency: 100000,
				Name:            renterAntName,
			},
		},
		AutoConnect: true,
		WaitForSync: true,
	}
	return config
}

// verify that createAntfarm() creates a new antfarm correctly.
func TestNewAntfarm(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	antFarmAddr := test.RandomLocalAddress()
	antAddr := test.RandomLocalAddress()
	dataDir := test.TestDir(t.Name())
	antFarmDir := filepath.Join(dataDir, "antfarm-data")
	antDirs := test.AntDirs(dataDir, 1)
	config := AntfarmConfig{
		ListenAddress: antFarmAddr,
		DataDir:       antFarmDir,
		AntConfigs: []ant.AntConfig{
			{
				SiadConfig: ant.SiadConfig{
					AllowHostLocalNetAddress: true,
					DataDir:                  antDirs[0],
					RPCAddr:                  antAddr,
					SiadPath:                 test.TestSiadFilename,
				},
				Jobs: []string{
					"gateway",
				},
			},
		},
	}

	antfarm, err := createAntfarm(config)
	if err != nil {
		t.Fatal(err)
	}
	defer antfarm.Close()

	go antfarm.ServeAPI()

	res, err := http.DefaultClient.Get("http://" + antFarmAddr + "/ants")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var ants []*ant.Ant
	err = json.NewDecoder(res.Body).Decode(&ants)
	if err != nil {
		t.Fatal(err)
	}
	if len(ants) != len(config.AntConfigs) {
		t.Fatal("expected /ants to return the correct number of ants")
	}
	if ants[0].RPCAddr != config.AntConfigs[0].RPCAddr {
		t.Fatal("expected /ants to return the correct rpc address")
	}
}

// verify that connectExternalAntfarm connects antfarms to eachother correctly
func TestConnectExternalAntfarm(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	datadir := test.TestDir(t.Name())
	antFarmDataDirs := []string{filepath.Join(datadir, "antfarm-data1"), filepath.Join(datadir, "antfarm-data2")}
	antConfig := ant.AntConfig{
		SiadConfig: ant.SiadConfig{
			AllowHostLocalNetAddress: true,
			RPCAddr:                  test.RandomLocalAddress(),
			SiadPath:                 test.TestSiadFilename,
		},
		Jobs: []string{
			"gateway",
		},
	}

	antConfig.SiadConfig.DataDir = test.AntDirs(antFarmDataDirs[0], 1)[0]
	config1 := AntfarmConfig{
		ListenAddress: test.RandomLocalAddress(),
		DataDir:       antFarmDataDirs[0],
		AntConfigs:    []ant.AntConfig{antConfig},
	}

	antConfig.SiadConfig.DataDir = test.AntDirs(antFarmDataDirs[1], 1)[0]
	antConfig.RPCAddr = test.RandomLocalAddress()
	config2 := AntfarmConfig{
		ListenAddress: test.RandomLocalAddress(),
		DataDir:       antFarmDataDirs[1],
		AntConfigs:    []ant.AntConfig{antConfig},
	}

	farm1, err := createAntfarm(config1)
	if err != nil {
		t.Fatal(err)
	}
	defer farm1.Close()
	go farm1.ServeAPI()

	farm2, err := createAntfarm(config2)
	if err != nil {
		t.Fatal(err)
	}
	defer farm2.Close()
	go farm2.ServeAPI()

	err = farm1.connectExternalAntfarm(config2.ListenAddress)
	if err != nil {
		t.Fatal(err)
	}

	// give a bit of time for the connection to succeed
	time.Sleep(time.Second * 3)

	// verify that farm2 has farm1 as its peer
	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = farm1.ants[0].APIAddr
	c := client.New(opts)
	gatewayInfo, err := c.GatewayGet()
	if err != nil {
		t.Fatal(err)
	}

	for _, ant := range farm2.ants {
		hasAddr := false
		for _, peer := range gatewayInfo.Peers {
			if fmt.Sprint(peer.NetAddress) == ant.RPCAddr {
				hasAddr = true
				break
			}
		}
		if !hasAddr {
			t.Fatalf("farm1 is missing %v", ant.RPCAddr)
		}
	}
}

// TestUploadDownloadFileData uploads and downloads a file and checks that
// their content is identical by comparing their merkle root hashes
func TestUploadDownloadFileData(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Start Antfarm
	dataDir := test.TestDir(t.Name())
	config := createBasicRenterAntfarmConfig(dataDir)
	farm, err := createAntfarm(config)
	if err != nil {
		t.Fatal(err)
	}
	defer farm.Close()

	// Timeout the test if the renter doesn't becomes upload ready
	renterAnt, err := farm.GetAntByName(renterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Upload a file
	renterJob := renterAnt.Jr.NewRenterJob()
	siaPath, err := renterJob.Upload(modules.SectorSize)
	if err != nil {
		t.Fatal(err)
	}

	// Download the file
	destPath := filepath.Join(renterAnt.Config.DataDir, "downloadedFiles", "downloadedFile")
	err = renterJob.Download(siaPath, destPath)
	if err != nil {
		t.Fatal(err)
	}

	// Compare file sizes
	sourceFilePath := filepath.Join("/", siaPath.Path)
	sourceFileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		t.Fatal(err)
	}
	downloadedFileInfo, err := os.Stat(destPath)
	if err != nil {
		t.Fatal(err)
	}
	sourceFileSize := sourceFileInfo.Size()
	downloadedFileSize := downloadedFileInfo.Size()
	if sourceFileSize != downloadedFileSize {
		t.Errorf("Source file size %v doesn't equal downloaded file size %v", sourceFileSize, downloadedFileSize)
	}

	// Compare file hashes
	file, err := os.Open(sourceFilePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sourceFileHash, err := ant.MerkleRoot(file)
	if err != nil {
		t.Fatal(err)
	}
	downloadedFileHash := renterJob.Files[0].MerkleRoot
	if sourceFileHash != downloadedFileHash {
		t.Error("Source file hash doesn't equal downloaded file hash")
	}
}

// TestUpdateRenter verifies that renter ant's siad can be upgraded using given
// path to siad binary
func TestUpdateRenter(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Start Antfarm
	dataDir := test.TestDir(t.Name())
	config := createBasicRenterAntfarmConfig(dataDir)
	farm, err := createAntfarm(config)
	if err != nil {
		t.Fatal(err)
	}
	defer farm.Close()

	// Timeout the test if the renter doesn't become upload ready
	renterAnt, err := farm.GetAntByName(renterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Restart the renter with given siad path (simulates an ant update
	// process)
	err = renterAnt.UpdateSiad(test.RelativeSiadPath())
	if err != nil {
		t.Fatal(err)
	}

	// Timeout the test if the renter after update doesn't become upload ready
	renterAnt, err = farm.GetAntByName(renterAntName)
	if err != nil {
		t.Fatal(err)
	}
	err = renterAnt.Jr.WaitForRenterUploadReady()
	if err != nil {
		t.Fatal(err)
	}

	// Verify that renter is working correctly by uploading and downloading a
	// file

	// Upload a file
	renterJob := renterAnt.Jr.NewRenterJob()
	siaPath, err := renterJob.Upload(modules.SectorSize)
	if err != nil {
		t.Fatal(err)
	}

	// Download the file
	destPath := filepath.Join(renterAnt.Config.DataDir, "downloadedFiles", "downloadedFile")
	err = renterJob.Download(siaPath, destPath)
	if err != nil {
		t.Fatal(err)
	}
}
