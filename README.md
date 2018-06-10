# cgschaincode
### Testing Using dev mode：
Navigate to the chaincode-docker-devmode directory of the fabric-samples clone:

`cd chaincode-docker-devmode`

##### Terminal 1 - Start the network
`docker-compose -f docker-compose-simple.yaml up`
##### Terminal 2 - Build & start the chaincode
`docker exec -it chaincode bash`

`cd cgschaincode`

`go build`

`CORE_PEER_ADDRESS=peer:7052 CORE_CHAINCODE_ID_NAME=mycc:0 ./cgschaincode`
##### Terminal 3 - Use the chaincode
`docker exec -it cli bash`

`peer chaincode install -p chaincodedev/chaincode/cgschaincode -n mycc -v 0`

`peer chaincode instantiate -n mycc -v 0 -c '{"Args":[""]}' -C myc`

##### Upgrade with the new version 1.0
`CORE_PEER_ADDRESS=peer:7052 CORE_CHAINCODE_ID_NAME=mycc:1 ./cgschaincode`

`peer chaincode install -p chaincodedev/chaincode/cgschaincode -n mycc -v 1`

`peer chaincode upgrade -n mycc2 -v 1 -c '{"Args":[""]}' -C myc`


### Security Chaincode Functions：
1. querySecurity(APIstub, args)
1. initLedger(APIstub, args)
1. createSecurity(APIstub, args)
1. queryAllSecurities(APIstub, args)
1. querySecurityStatus(APIstub, args)
1. queryOwner(APIstub, args)
1. queryOwnerAccount(APIstub, args)
1. queryOwnerLength(APIstub, args)
1. changeSecurity(APIstub, args)
1. changeSecurityStatus(APIstub, args)
1. changeOwnerAvaliable(APIstub, args)
1. deleteSecurity(APIstub, args)
1. deleteOwner(APIstub, args)
1. updateOwnerInterest(APIstub, args)
1. getHistoryForSecurity(APIstub, args)
1. getHistoryTXIDForSecurity(APIstub, args)
1. queryAllSecurityKeys(APIstub, args)
1. changeBankSecurityTotals(APIstub, args)
1. queryBankSecurityTotals(APIstub, args)


### Account Chaincode Functions
1. initAccount(APIstub, args)
1. deleteAccount(APIstub, args)
1. getStateAsBytes(APIstub, args)
1. updateAccountStatus(APIstub, args)
1. updateAccount(APIstub, args)
1. updateAsset(APIstub, args)
1. updateAssetBalance(APIstub, args)
1. deleteAsset(APIstub, args)
1. queryAsset(APIstub, args)
1. queryAssetInfo(APIstub, args)
1. queryAssetLength(APIstub, args)
1. queryAccountStatus(APIstub, args)
1. queryAllAccounts(APIstub, args)
1. getHistoryForAccount(APIstub, args)
1. getHistoryTXIDForAccount(APIstub, args)
1. queryAllAccountKeys(APIstub, args)


### Bank Chaincode Functions
1. initBank(APIstub, args)
1. updateBank(APIstub, args)
1. deleteBank(APIstub, args)
1. verifyBankList(APIstub, args)
1. getStateAsBytes(APIstub, args)
1. queryAllBanks(APIstub, args)
1. getHistoryForBank(APIstub, args)
1. getHistoryTXIDForBank(APIstub, args)
1. queryAllBankKeys(APIstub, args)


### Transaction Chaincode Functions
1. submitApproveTransaction(APIstub, args)
1. submitEndDayTransaction(APIstub, args)
1. securityTransfer(APIstub, args)
1. securityCorrectTransfer(APIstub, args)
1. queryTXIDTransactions(APIstub, args)
1. queryTXKEYTransactions(APIstub, args)
1. queryHistoryTXKEYTransactions(APIstub, args)
1. getHistoryForTransaction(APIstub, args)
1. getHistoryTXIDForTransaction(APIstub, args)
1. getHistoryForQueuedTransaction(APIstub, args)
1. getHistoryTXIDForQueuedTransaction(APIstub, args)
1. queryAllTransactions(APIstub, args)
1. queryAllQueuedTransactions(APIstub, args)
1. queryAllHistoryTransactions(APIstub, args)
1. queryAllTransactionKeys(APIstub, args)
1. queryQueuedTransactionStatus(APIstub, args)
1. queryHistoryTransactionStatus(APIstub, args)


### Other Chaincode Functions
1. mapFunction(APIstub, function, args)
1. get(APIstub, function, args)
1. put(APIstub, function, args)
1. remove(APIstub, function, args)
1. keys(APIstub, function, args)
1. query(APIstub, function, args)