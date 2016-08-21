// Copyright (c) 2015 The Decred Developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	//"bufio"
	//"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	//"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/decred/dcrd/chaincfg"
	"github.com/decred/dcrd/chaincfg/chainec"
	"github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/decred/dcrutil"
	"github.com/decred/dcrutil/hdkeychain"
	//"github.com/decred/dcrwallet/pgpwordlist"
)

// The hierarchy described by BIP0043 is:
//  m/<purpose>'/*
// This is further extended by BIP0044 to:
//  m/44'/<coin type>'/<account>'/<branch>/<address index>
//
// The branch is 0 for external addresses and 1 for internal addresses.

// maxCoinType is the maximum allowed coin type used when structuring
// the BIP0044 multi-account hierarchy.  This value is based on the
// limitation of the underlying hierarchical deterministic key
// derivation.
const maxCoinType = hdkeychain.HardenedKeyStart - 1

// MaxAccountNum is the maximum allowed account number.  This value was
// chosen because accounts are hardened children and therefore must
// not exceed the hardened child range of extended keys and it provides
// a reserved account at the top of the range for supporting imported
// addresses.
const MaxAccountNum = hdkeychain.HardenedKeyStart - 2 // 2^31 - 2

// ExternalBranch is the child number to use when performing BIP0044
// style hierarchical deterministic key derivation for the external
// branch.
const ExternalBranch uint32 = 0

// InternalBranch is the child number to use when performing BIP0044
// style hierarchical deterministic key derivation for the internal
// branch.
const InternalBranch uint32 = 1

var curve = secp256k1.S256()

var params = chaincfg.MainNetParams

// Flag arguments.
var getHelp = flag.Bool("h", false, "Print help message")
var testnet = flag.Bool("testnet", false, "")
var simnet = flag.Bool("simnet", false, "")

// var noseed = flag.Bool("noseed", false, "Generate a single keypair instead of "+
// 	"an HD extended seed")
var verify = flag.Bool("verify", false, "Verify a seed by generating the first "+
	"address")
var pattern1 = flag.String("pattern1", "", "Primary pattern. dcrvanity will exit if this matches.")
var pattern2 = flag.String("pattern2", "", "Secondary pattern. dcrvanity will NOT exit if this matches.")

func setupFlags(msg func(), f *flag.FlagSet) {
	f.Usage = msg
}

var newLine = "\n"

// writeNewFile writes data to a file named by filename.
// Error is returned if the file does exist. Otherwise writeNewFile creates the file with permissions perm;
// Based on ioutil.WriteFile, but produces an err if the file exists.
func writeNewFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		// There was no error, but not all the data was written, so report an error.
		err = io.ErrShortWrite
	}
	if err == nil {
		// There was an error, so close file (ignoreing any further errors) and return the error.
		f.Close()
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

// searchKeyPair generates a secp256k1 keypair
func searchKeyPair(primaryPattern, secondaryPattern string, inclusive bool) (*secp256k1.PrivateKey, *dcrutil.AddressPubKeyHash, error) {
	var regexPrimary, regexSecondary *regexp.Regexp
	var err error

	// pat1 := "chap(?i)p"
	if len(secondaryPattern) > 0 {
		regexSecondary, err = regexp.Compile("^Ds" + secondaryPattern)
		if err != nil {
			return nil, nil, err // there was a problem with the regular expression.
		}
	} else if inclusive {
		fmt.Println("nil secondary pattern and inclusive is true. No addresses will be checked.")
		return nil, nil, err
	}
	fmt.Println("Secondary pattern: ", regexSecondary.String())

	// pat2 := "chapp(?i)jc"
	if len(primaryPattern) > 0 {
		regexPrimary, err = regexp.Compile("^Ds" + primaryPattern)
		if err != nil {
			return nil, nil, err // there was a problem with the regular expression.
		}
	} else {
		fmt.Println("nil primary pattern. The program will never quit.")
	}
	fmt.Println("Primary pattern: ", regexPrimary.String())

	var key *ecdsa.PrivateKey
	var addr *dcrutil.AddressPubKeyHash

	for i := 0; ; i++ {
		key0, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pub := secp256k1.PublicKey{
			Curve: curve,
			X:     key0.PublicKey.X,
			Y:     key0.PublicKey.Y,
		}

		addr0, err := dcrutil.NewAddressPubKeyHash(Hash160(pub.SerializeCompressed()),
			&params, chainec.ECTypeSecp256k1)
		if err != nil {
			return nil, nil, err
		}

		if i%10000 == 0 {
			fmt.Printf("\r%d", i)
		}

		// If the secondary pattern is matched by the primary, this is faster
		// because if the secondary does't match, then neither will primary.
		if inclusive {
			if regexSecondary != nil && regexSecondary.MatchString(addr0.EncodeAddress()) {
				fmt.Printf("\r%d\n%s\n", i, addr0.EncodeAddress())
				fmt.Printf("%x\n", pub.SerializeCompressed())

				privX := secp256k1.PrivateKey{
					PublicKey: key0.PublicKey,
					D:         key0.D,
				}
				//privWifX := NewWIF(privX)
				//fmt.Printf("%s\n", privWifX.String())
				fmt.Println(privX)

				if regexPrimary != nil && regexPrimary.MatchString(addr0.EncodeAddress()) {
					key = key0
					addr = addr0
					fmt.Printf("Woohoo!\n")
					break
				}
			}
		} else {
			// primary match does not imply secondary, so check both separately
			if regexSecondary != nil && regexSecondary.MatchString(addr0.EncodeAddress()) {
				fmt.Printf("\r%d\n%s\n", i, addr0.EncodeAddress())
				fmt.Printf("%x\n", pub.SerializeCompressed())

				privX := secp256k1.PrivateKey{
					PublicKey: key0.PublicKey,
					D:         key0.D,
				}
				privWifX := NewWIF(privX)
				fmt.Printf("%s\n", privWifX.String())
			}

			if regexPrimary != nil && regexPrimary.MatchString(addr0.EncodeAddress()) {
				fmt.Printf("Woohoo!\n")
				fmt.Printf("\r%d\n%s\n", i, addr0.EncodeAddress())
				fmt.Printf("%x\n", pub.SerializeCompressed())
				key = key0
				addr = addr0

				privX := secp256k1.PrivateKey{
					PublicKey: key0.PublicKey,
					D:         key0.D,
				}
				privWifX := NewWIF(privX)
				fmt.Printf("%s\n", privWifX.String())

				break
			}
		}
	}

	priv := &secp256k1.PrivateKey{
		PublicKey: key.PublicKey,
		D:         key.D,
	}

	return priv, addr, nil

	//privWif := NewWIF(priv)

	// var buf bytes.Buffer
	// buf.WriteString("Address: ")
	// buf.WriteString(addr.EncodeAddress())
	// buf.WriteString(" | ")
	// buf.WriteString("Private key: ")
	// buf.WriteString(privWif.String())
	//buf.WriteString(newLine)

	// outs := buf.String()
	// fmt.Println(outs)

	// err = writeNewFile(filename, buf.Bytes(), 0600)
	// if err != nil {
	// return err
	// }
	//return nil
}

// deriveCoinTypeKey derives the cointype key which can be used to derive the
// extended key for an account according to the hierarchy described by BIP0044
// given the coin type key.
//
// In particular this is the hierarchical deterministic extended key path:
// m/44'/<coin type>'
func deriveCoinTypeKey(masterNode *hdkeychain.ExtendedKey,
	coinType uint32) (*hdkeychain.ExtendedKey, error) {
	// Enforce maximum coin type.
	if coinType > maxCoinType {
		return nil, fmt.Errorf("bad coin type")
	}

	// The hierarchy described by BIP0043 is:
	//  m/<purpose>'/*
	// This is further extended by BIP0044 to:
	//  m/44'/<coin type>'/<account>'/<branch>/<address index>
	//
	// The branch is 0 for external addresses and 1 for internal addresses.

	// Derive the purpose key as a child of the master node.
	purpose, err := masterNode.Child(44 + hdkeychain.HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	// Derive the coin type key as a child of the purpose key.
	coinTypeKey, err := purpose.Child(coinType + hdkeychain.HardenedKeyStart)
	if err != nil {
		return nil, err
	}

	return coinTypeKey, nil
}

// deriveAccountKey derives the extended key for an account according to the
// hierarchy described by BIP0044 given the master node.
//
// In particular this is the hierarchical deterministic extended key path:
//   m/44'/<coin type>'/<account>'
func deriveAccountKey(coinTypeKey *hdkeychain.ExtendedKey,
	account uint32) (*hdkeychain.ExtendedKey, error) {
	// Enforce maximum account number.
	if account > MaxAccountNum {
		return nil, fmt.Errorf("account num too high")
	}

	// Derive the account key as a child of the coin type key.
	return coinTypeKey.Child(account + hdkeychain.HardenedKeyStart)
}

// checkBranchKeys ensures deriving the extended keys for the internal and
// external branches given an account key does not result in an invalid child
// error which means the chosen seed is not usable.  This conforms to the
// hierarchy described by BIP0044 so long as the account key is already derived
// accordingly.
//
// In particular this is the hierarchical deterministic extended key path:
//   m/44'/<coin type>'/<account>'/<branch>
//
// The branch is 0 for external addresses and 1 for internal addresses.
func checkBranchKeys(acctKey *hdkeychain.ExtendedKey) error {
	// Derive the external branch as the first child of the account key.
	if _, err := acctKey.Child(ExternalBranch); err != nil {
		return err
	}

	// Derive the external branch as the second child of the account key.
	_, err := acctKey.Child(InternalBranch)
	return err
}

func main() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
	helpMessage := func() {
		fmt.Println("Usage: dcraddrgen [-testnet] [-simnet] [-h] filename")
		fmt.Println("Generate a Decred private and public key, with address matching pattern(s).")
		//"These are output to the file 'filename'.\n")
		fmt.Println("  -h \t\tPrint this message")
		fmt.Println("  -testnet \tGenerate a testnet key instead of mainnet")
		fmt.Println("  -simnet \tGenerate a simnet key instead of mainnet")
		fmt.Println("  -pattern1 \tPrimary pattern. dcrvanity will exit if this matches.")
		fmt.Println("  -pattern2 \tSecondary pattern. dcrvanity will NOT exit if this matches.")
	}

	setupFlags(helpMessage, flag.CommandLine)
	flag.Parse()

	if *getHelp {
		helpMessage()
		return
	}

	// var fileName string
	// if flag.Arg(0) != "" {
	// 	fileName = flag.Arg(0)
	// } else {
	// 	fileName = "keys.txt"
	// }

	// Alter the globals to specified network.
	if *testnet {
		if *simnet {
			fmt.Println("Error: Only specify one network.")
			return
		}
		params = chaincfg.TestNetParams
	}
	if *simnet {
		params = chaincfg.SimNetParams
	}

	// Single keypair generation/search
	priv, addr, err := searchKeyPair(*pattern1, *pattern2, true)
	if err != nil {
		fmt.Printf("Error generating key pair: %v\n", err.Error())
		return
	}

	spew.Dump(priv, addr)
	privWif := NewWIF(*priv)
	fmt.Printf("%v\n%s\n", privWif, privWif.String())

	// fmt.Printf("Successfully generated keypair and stored it in %v.\n",
	// fn)
	// fmt.Printf("Your private key is used to spend your funds. Do not " +
	// "reveal it to anyone.\n")
	return

}