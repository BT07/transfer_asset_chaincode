package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	//"strconv"
	"strings"
	//"time"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	mspprotos "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

type member struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Role    string `json:"role"`
}

type asset struct {
	AssetName string `json:"assetname"`
	Creator   string `json:"creator"`
	Current   string `json:"current"`
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response { //Nice
	return shim.Success(nil)
}

func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)

	creator, err := stub.GetCreator() // it'll give the certificate of the invoker
	id := &mspprotos.SerializedIdentity{}
	err = proto.Unmarshal(creator, id)
	if err != nil {
		return shim.Error(fmt.Sprintf("chaincode::AcceptLeadQuote:couldnt unmarshal creator"))
	}
	block, _ := pem.Decode(id.GetIdBytes())
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return shim.Error(fmt.Sprintf("chaincode:AcceptLeadQuote:couldnt parse certificate"))
	}
	invokerhash := sha256.Sum256([]byte(cert.Subject.CommonName + cert.Issuer.CommonName))
	insurerAddress := hex.EncodeToString(invokerhash[:])

	// Handle different functions
	if function == "addMember" {
		return t.addMember(stub, args, insurerAddress, string(cert.Subject.CommonName))
	} else if function == "readMember" { //read a member
		return t.readMember(stub, insurerAddress)
	} else if function == "addAsset" { //add a asset
		return t.addAsset(stub, args, insurerAddress)
	} else if function == "readAsset" { //read a asset
		return t.readAsset(stub, args)
	} else if function == "transferAsset" { //read a asset
		return t.transferAsset(stub, args, insurerAddress)
	}

	fmt.Println("invoke did not find func: " + function) //error
	return shim.Error("Received unknown function invocation")
}

func (t *SimpleChaincode) addMember(stub shim.ChaincodeStubInterface, args []string, insurerAddress string, insurername string) pb.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	// ==== Input sanitation ====
	fmt.Println("- start init marble")
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("2nd argument must be a non-empty string")
	}

	if len(insurerAddress) <= 0 {
		return shim.Error("Invalid peer ")
	}

	clientName := insurername
	Address := strings.ToLower(args[0])
	Role := strings.ToLower(args[1])

	insurerAsBytes, err := stub.GetState(insurerAddress)
	if err != nil {
		return shim.Error("Failed to get client: " + err.Error())
	} else if insurerAsBytes != nil {
		fmt.Println("This client already exists: " + insurerAddress)
		return shim.Error("This client already exists: " + insurerAddress)
	}

	member := &member{insurerAddress, clientName, Address, Role}
	memberJSONasBytes, err := json.Marshal(member)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(insurerAddress, memberJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("This is the adderess of invoker :", insurerAddress, "|-------------------", []byte(insurerAddress), "--------sd-s----")
	return shim.Success([]byte(insurerAddress))
}

func (t *SimpleChaincode) readMember(stub shim.ChaincodeStubInterface, arg string) pb.Response {
	var name, jsonResp string
	var err error

	if len(arg) <= 0 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = arg
	valAsbytes, err := stub.GetState(name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Member does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(valAsbytes)
}

func (t *SimpleChaincode) addAsset(stub shim.ChaincodeStubInterface, args []string, invokerid string) pb.Response {
	var err error
	var jsonResp string
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	// ==== Input sanitation ====
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}

	valAsBytes, err := stub.GetState(invokerid)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + invokerid + "\"}"
		return shim.Error(jsonResp)
	} else if valAsBytes == nil {
		jsonResp = "{\"Error\":\"Member does not exist: " + invokerid + "\"}"
		return shim.Error(jsonResp)
	}
	memberobject := member{}
	err = json.Unmarshal(valAsBytes, &memberobject) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	if memberobject.Role == "manufacturer" {
		assetname := args[0]

		assetobj := &asset{assetname, invokerid, invokerid}
		assetJSONasBytes, err := json.Marshal(assetobj)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(assetname, assetJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else {
		return shim.Error("Not a manufacturer")
	}
	return shim.Success(valAsBytes)
}

func (t *SimpleChaincode) readAsset(stub shim.ChaincodeStubInterface, arg []string) pb.Response {
	var name, jsonResp string
	var err error

	if len(arg[0]) <= 0 {
		return shim.Error("Incorrect number of arguments. Expecting name of the marble to query")
	}

	name = arg[0]
	valAsbytes, err := stub.GetState(name)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return shim.Error(jsonResp)
	} else if valAsbytes == nil {
		jsonResp = "{\"Error\":\"Member does not exist: " + name + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(valAsbytes)
}

func (t *SimpleChaincode) transferAsset(stub shim.ChaincodeStubInterface, args []string, invokerid string) pb.Response {
	var err error
	var jsonResp string
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	// ==== Input sanitation ====
	if len(args[0]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("1st argument must be a non-empty string")
	}

	valAsBytes, err := stub.GetState(args[1])
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + args[1] + "\"}"
		return shim.Error(jsonResp)
	} else if valAsBytes == nil {
		jsonResp = "{\"Error\":\"Member does not exist: " + args[1] + "\"}"
		return shim.Error(jsonResp)
	}

	assetvalAsBytes, err := stub.GetState(args[0])
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + args[0] + "\"}"
		return shim.Error(jsonResp)
	} else if assetvalAsBytes == nil {
		jsonResp = "{\"Error\":\"Asset does not exist: " + args[0] + "\"}"
		return shim.Error(jsonResp)
	}
	assetobject := asset{}
	err = json.Unmarshal(assetvalAsBytes, &assetobject) //unmarshal it aka JSON.parse()
	if err != nil {
		return shim.Error(err.Error())
	}
	if invokerid == assetobject.Current {
		assetcurrent := args[1]

		assetobj := &asset{assetobject.AssetName, assetobject.Creator, assetcurrent}
		assetJSONasBytes, err := json.Marshal(assetobj)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = stub.PutState(assetobject.AssetName, assetJSONasBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	} else {
		return shim.Error("Not a current owner")
	}
	return shim.Success([]byte("Asset Transferred"))
}
