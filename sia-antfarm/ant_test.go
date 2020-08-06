package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"gitlab.com/NebulousLabs/Sia-Ant-Farm/ant"
	"gitlab.com/NebulousLabs/Sia-Ant-Farm/test"
	"gitlab.com/NebulousLabs/Sia/node/api/client"
)

// TestStartAnts verifies that startAnts successfully starts ants given some
// configs.
func TestStartAnts(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs := test.AntDirs(dataDir, 3)
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadPath,
			},
		},
	}

	// Start ants
	ants, err := startAnts(&sync.WaitGroup{}, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			ant.Close()
		}
	}()

	// verify each ant has a reachable api
	for _, ant := range ants {
		opts, err := client.DefaultOptions()
		if err != nil {
			t.Fatal(err)
		}
		opts.Address = ant.APIAddr
		c := client.New(opts)
		if _, err := c.ConsensusGet(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestStartAntWithSiadPath verifies that startAnts successfully starts ant
// given relative or absolute path to siad binary that is not in PATH
func TestStartAntWithSiadPath(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Paths to binaries are different in local environment and in Gitlab CI/CD
	var relativeSiadPath string
	if _, ok := os.LookupEnv("GITLAB_CI"); ok {
		// In Gitlab CI/CD
		relativeSiadPath = ".cache/bin/siad-dev"
	} else {
		// Locally
		relativeSiadPath = "../../../../../bin/siad-dev"
	}
	absoluteSiadPath, err := filepath.Abs(relativeSiadPath)
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		name     string
		siadPath string
	}{
		{name: "TestRelativePath", siadPath: relativeSiadPath},
		{name: "TestAbsolutePath", siadPath: absoluteSiadPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimum configs
			dataDir := test.TestDir(tt.name)
			antDirs := test.AntDirs(dataDir, 1)
			configs := []ant.AntConfig{
				{
					SiadConfig: ant.SiadConfig{
						AllowHostLocalNetAddress: true,
						DataDir:                  antDirs[0],
						SiadPath:                 tt.siadPath,
					},
				},
			}

			// Start an ant
			ants, err := startAnts(&sync.WaitGroup{}, configs...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				for _, ant := range ants {
					ant.Close()
				}
			}()

			// Verify the ant has a reachable api
			for _, ant := range ants {
				opts, err := client.DefaultOptions()
				if err != nil {
					t.Fatal(err)
				}
				opts.Address = ant.APIAddr
				c := client.New(opts)
				if _, err := c.ConsensusGet(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// TestRenterDisableIPViolationCheck verifies that IPViolationCheck can be set
// via renter ant config
func TestRenterDisableIPViolationCheck(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Define test cases data
	testCases := []struct {
		name                          string
		dataDirPostfix                string
		renterDisableIPViolationCheck bool
	}{
		{"TestDefaultIPViolationCheck", "-default", false},
		{"TestDisabledIPViolationCheck", "-ip-check-disabled", true},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create minimum configs
			dataDir := test.TestDir(t.Name() + tc.dataDirPostfix)
			antDirs := test.AntDirs(dataDir, 1)
			configs := []ant.AntConfig{
				{
					SiadConfig: ant.SiadConfig{
						AllowHostLocalNetAddress: true,
						DataDir:                  antDirs[0],
						SiadPath:                 test.TestSiadPath,
					},
					Jobs: []string{"renter"},
				},
			}

			// Update config if testing disabled IP violation check
			if tc.renterDisableIPViolationCheck {
				configs[0].RenterDisableIPViolationCheck = true
			}

			// Start ant
			ants, err := startAnts(&sync.WaitGroup{}, configs...)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				for _, ant := range ants {
					ant.Close()
				}
			}()
			renterAnt := ants[0]

			// Get http client
			c, err := getClient(renterAnt.APIAddr, "")
			if err != nil {
				t.Fatal(err)
			}

			// Get renter settings
			renterInfo, err := c.RenterGet()
			if err != nil {
				t.Fatal(err)
			}
			// Check that IP violation check was not set by default and was set
			// correctly if configured so
			if !tc.renterDisableIPViolationCheck && !renterInfo.Settings.IPViolationCheck {
				t.Fatal("Setting IPViolationCheck is supposed to be true by default")
			} else if tc.renterDisableIPViolationCheck && renterInfo.Settings.IPViolationCheck {
				t.Fatal("Setting IPViolationCheck is supposed to be set false by the ant config")
			}
		})
	}
}

// TestConnectAnts verifies that ants will connect
func TestConnectAnts(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// connectAnts should throw an error if only one ant is provided
	if err := connectAnts(&ant.Ant{}); err == nil {
		t.Fatal("connectAnts didnt throw an error with only one ant")
	}

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs := test.AntDirs(dataDir, 5)
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[3],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[4],
				SiadPath:                 test.TestSiadPath,
			},
		},
	}

	// Start ants
	ants, err := startAnts(&sync.WaitGroup{}, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			ant.Close()
		}
	}()

	// Connect the ants
	err = connectAnts(ants...)
	if err != nil {
		t.Fatal(err)
	}

	// Get the Gateway info from on of the ants
	opts, err := client.DefaultOptions()
	if err != nil {
		t.Fatal(err)
	}
	opts.Address = ants[0].APIAddr
	c := client.New(opts)
	gatewayInfo, err := c.GatewayGet()
	if err != nil {
		t.Fatal(err)
	}
	// Verify the ants are peers
	for _, ant := range ants[1:] {
		hasAddr := false
		for _, peer := range gatewayInfo.Peers {
			if fmt.Sprint(peer.NetAddress) == ant.RPCAddr {
				hasAddr = true
				break
			}
		}
		if !hasAddr {
			t.Fatalf("the central ant is missing %v", ant.RPCAddr)
		}
	}
}

// TestAntConsensusGroups probes the antConsensusGroup function
func TestAntConsensusGroups(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	t.Parallel()

	// Create minimum configs
	dataDir := test.TestDir(t.Name())
	antDirs := test.AntDirs(dataDir, 4)
	configs := []ant.AntConfig{
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[0],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[1],
				SiadPath:                 test.TestSiadPath,
			},
		},
		{
			SiadConfig: ant.SiadConfig{
				AllowHostLocalNetAddress: true,
				DataDir:                  antDirs[2],
				SiadPath:                 test.TestSiadPath,
			},
		},
	}

	// Start Ants
	ants, err := startAnts(&sync.WaitGroup{}, configs...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, ant := range ants {
			ant.Close()
		}
	}()

	// Get the consensus groups
	groups, err := antConsensusGroups(ants...)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 {
		t.Fatal("expected 1 consensus group initially")
	}
	if len(groups[0]) != len(ants) {
		t.Fatal("expected the consensus group to have all the ants")
	}

	// Start an ant that is desynced from the rest of the network
	cfg, err := parseConfig(ant.AntConfig{
		Jobs: []string{"miner"},
		SiadConfig: ant.SiadConfig{
			AllowHostLocalNetAddress: true,
			DataDir:                  antDirs[3],
			SiadPath:                 test.TestSiadPath,
		},
	},
	)
	if err != nil {
		t.Fatal(err)
	}
	otherAnt, err := ant.New(&sync.WaitGroup{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	ants = append(ants, otherAnt)

	// Wait for the other ant to mine a few blocks
	time.Sleep(time.Second * 30)

	// Verify the ants are synced
	groups, err = antConsensusGroups(ants...)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatal("expected 2 consensus groups")
	}
	if len(groups[0]) != len(ants)-1 {
		t.Fatal("expected the first consensus group to have 3 ants")
	}
	if len(groups[1]) != 1 {
		t.Fatal("expected the second consensus group to have 1 ant")
	}
	if !reflect.DeepEqual(groups[1][0], otherAnt) {
		t.Fatal("expected the miner ant to be in the second consensus group")
	}
}
