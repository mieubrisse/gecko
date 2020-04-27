# WASM Smart Contract VM

This Virtual Machine implements a chain (the "WASM chain") that acts as a platform for WASM smart contracts.

It allows users to upload and invoke smart contracts.

In this branch, the WASM chain is included in the network genesis and is validated by the Default Subnet.
That is, it's "built-in"; you need not launch a subnet or the chain.

To launch a one node network:

``` sh
./build/ava --snow-sample-size=1 --snow-quorum-size=1 --network-id=local --staking-tls-enabled=false --public-ip=127.0.0.1 --log-level=debug
```

## A Sample Smart Contract

See [here](https://github.com/ava-labs/gecko-internal/blob/wasm/vms/wasmvm/contracts/rust_bag/src/lib.rs) for a Rust implementation of a smart contract.

In this smart contract there are 2 entities: bags and owners.

A bag has a handful of properties, such as price.
Each bag has an owner, and each owner has 0 or more bags.

The smart contract allows for the creation of owners and bags, the transfer of bags between owners, and the update of bag prices.
All data is stored in two global variables, `OWNERS` and `BAGS`.

To compile the Rust code to WASM, we use the [wasm-pack](https://rustwasm.github.io/wasm-pack/installer/) tool in the Rust code's directory:

```sh
wasm-pack build
```

## Creating a Private Key

The WASM chain uses an account-based model.
Each transaction on the chain is _sent_ by an account.
Each account corresponds to a private key, which is used to sign transactions sent by the account.
Each account has a nonce, which is incremented every time the account sends a transaction.
Before we send any transactions, we need a new private key.
To get one, we call `wasm.newKey`:

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.newKey",
    "params": {},
    "id": 1
}'
```

The response contains a new private key, which we'll use to sign transaction on-chain transactions.

```json
{
    "jsonrpc": "2.0",
    "result": {
        "privateKey": "RuuQbE5FPNwDHmkCbu9oxgR35eNFgb2eVkh6TLw7qbgqp5mn6"
    },
    "id": 1
}
```

## Uploading a Smart Contract

To upload a smart contract, call `wasm.createContract`.
This API method takes an argument, `contract`, which is the base-58 with checksum representation of a WASM file.
It also takes arguments `senderKey` and `senderNonce`, which are the private key and nonce of the account sending the transaction.
Below, we create an instance of the contract we defined above.
`senderKey` is the key we generated above, and `senderNonce` is 1 since this is the first transaction sent by our account.
Since the smart contract is ~4 Kilobytes, we omit it below so as to not take up the whole page.

Sample call:

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.createContract",
    "params": {
        "senderKey":"RuuQbE5FPNwDHmkCbu9oxgR35eNFgb2eVkh6TLw7qbgqp5mn6",
    	"senderNonce":1,
        "contract": "CONTRACT BYTES GO HERE"
    },
    "id": 1
}'
```

This method returns the ID of the generated transaction, which is also the ID of the smart contract:

```json
{
    "jsonrpc": "2.0",
    "result": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
    "id": 1
}
```

## Getting Transaction Details

We can get details about a transaction by calling `wasm.getTx`.
This API method takes one argument, the transaction's ID.

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.getTx",
    "params": {
        "id": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j"
    },
    "id": 1
}'
```

For a contract creation transaction, this method returns the transaction's type and the transaction itself.

```json
{
    "jsonrpc": "2.0",
    "result": {
        "tx": {
            "tx": {
                "contract": "CONTRACT BYTES GO HERE",
                "id": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
                "nonce": "1",
                "sender": "MsfAd26DBEUsadYdeGrk6SH84fMgedmQD"
            },
            "type": "contract creation"
        }
    },
    "id": 1
}
```

## Invoking a Smart Contract

A smart contract that has been uploaded can be invoked by calling `wasm.invoke`.
This method takes six arguments:

* `contractID`: The contract being invoked
* `function`: The name of the function being invoked
* `senderKey` and `senderNonce`: Similar to previous call. Note that we increment `senderNonce` to 2.
* `args`: An array of integer arguments to pass to the function being invoked. May be omitted.
  * Each element of `args` must specify its type, which is one of: `int32`, `int64`
* `byteArgs`: The base 58 with checksum representation of a byte array to pass to the smart contract. May be omitted.

Let's invoke the contract's `create_owner` method.
As you can see in the Rust code, this method takes one argument: the owner's ID.
We give the owner ID 123 below.

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.invoke",
    "params":{
    	"contractID":"Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
    	"function":"create_owner",
        "senderNonce":"2",
        "senderKey":"RuuQbE5FPNwDHmkCbu9oxgR35eNFgb2eVkh6TLw7qbgqp5mn6",
        "args": [
            {
                "type": "int32",
                "value": 123
            }
        ]
    },
    "id": 1
}'
```

The resulting transaction's ID is returned:

```json
{
    "jsonrpc": "2.0",
    "result": {
        "txID": "d84yyAPCypq9JawDXEuntrfs8ggbBDAqbmEic7MM8cUa1E2Kb"
    },
    "id": 1
}
```

Now we can see this transaction's result by calling `wasm.getTx`:

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.getTx",
    "params": {
        "id": "d84yyAPCypq9JawDXEuntrfs8ggbBDAqbmEic7MM8cUa1E2Kb"
    },
    "id": 1
}'
```

The response indicates that the transaction was a contract invocation and that the method being called didn't return an error.
The response show the arguments to the invoked method as well as the returned value.
`byteArguments` and `returnValue` are both encoded with base 58 and a checksum.
In this case, both have value `"45PJLL"`, which is the encoding for an empty byte array.
That is, this method received no `byteArguments` and returned nothing (void.) 
`sender` is the _address_ of the account that sent the transaction.
This is the hash of the public key that corresponds to the private key that controls the account. 

```json
{
    "jsonrpc": "2.0",
    "result": {
        "tx": {
            "invocationSuccessful": true,
            "returnValue": "45PJLL",
            "tx": {
                "arguments": [
                    2
                ],
                "byteArgs": "45PJLL",
                "contractID": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
                "function": "create_owner",
                "id": "d84yyAPCypq9JawDXEuntrfs8ggbBDAqbmEic7MM8cUa1E2Kb",
                "sender": "MsfAd26DBEUsadYdeGrk6SH84fMgedmQD",
                "senderNonce": "2"
            },
            "type": "contract invocation"
        }
    },
    "id": 1
}
```

## Imported Functions

WASM smart contracts may (and in practice do) need to import information/functionality from the "outside world" (ie the WASM chain.)
The WASM chain provides an interface for the contracts to use.
Part of this interface is a key/value database.
Each contract has its own database that only it reads/writes.

Right now, the following methods are provided to contracts:

* `void print(int ptr, int len)`
    * Print to the chain's log
    * `ptr` is a pointer to the first element of a byte array
    * `len` is the byte array's length  
* `int dbPut(int keyPtr, int keyLen, int value, int valueLen)`
    * Put a key/value pair in the smart contract's database
    * `keyPtr` is a pointer to the first element of a byte array (the key.)
    * `keyLen` is the byte array's length  
    * Similar for `value` and `valueLen`
    * Returns 0 on success, otherwise non 0
* `int dbGet(int keyPtr, int keyLen, int value)`
    * Get a value from the smart contract's database.
    * `keyPtr` and `keyLen` specify the key.
    * `value` is a pointer to a buffer to write the value to.
    * Returns the length of the value, or -1 on failure.
* `int dbDelete(int keyPtr, int keyLen)`
    * Delete a key/value pair in the smart contract's database.
    * `keyPtr` is a pointer to the first element of a byte array (the key)
    * `keyLen` is the key's length 
    * Returns 0 on success, otherwise non-zero.
* `int dbGetValueLen(int keyPtr, int keyLen)`
    * Return the length of the value whose key is specified by `keyPtr` and `keyLen`
    * Return -1 on failure
* `int returnValue(int valuePtr, int valueLen)`
    * Called by the contract to return a value
    * The return value will be persisted
    * `valuePtr` is a pointer to the first element of a byte array (the return value.)
    * `valueLen` is the return value's length
    * Contract's that don't call this method are considered to return void.
    * If a contract calls this method multiple times, only the last value will be persisted.
    * Returns 0 on success, otherwise non-zero
* `int getArgs(int ptr)`
    * Get the byte arguments to this contract method invocation.
    * `ptr` is a pointer to a byte array. The args will be written here.
    * The args are guaranteed to be no longer than 1024.
    * Returns the length of the args, or -1 on failure.
* `int getSender(int ptr)`
    * Get the address of the sender of the transaction that invoked this contract method.
    * The address is 20 bytes.
    * The address will be written to the byte array pointed to by `ptr`.
    * Returns 0 on success, otherwise non-zero.
* `int getTxID(int ptr)`
    * Get the ID of the transaction that invoked this contract method.
    * Writes the ID, which is 32 bytes, to the byte array pointed to by `ptr`.
    * Returns 0 on success, otherwise non-zero.

## Calling Conventions

WASM methods can only take as arguments integers and floats, and can only return an integer or a float.
This model is rather restrictive, so we've created a calling convention to allow WASM smart contract methods to handle more expressive arguments and return values.

### Byte Arguments

One can pass a byte array to a contract method by providing argument `byteArgs` when calling `wasm.Invoke`.

You may be thinking, "wait, didn't you just say WASM methods can't take byte array arguments?" Well, that's true. We don't pass in the byte arguments directly. The contract can read the byte arguments by calling imported method `getArgs` (see "Imported Function" section above.) 

#### Example

The method `print_byte_args` in the contract we defined above reads the byte arguments to it, then uses `print` to print them.

When we call it:

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.invoke",
    "params": {
        "contractID": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
        "function": "print_byte_args",
        "senderNonce":"3",
        "senderKey":"RuuQbE5FPNwDHmkCbu9oxgR35eNFgb2eVkh6TLw7qbgqp5mn6",
        "args": [],
        "byteArgs":"U1Gavwb6Dr7nwea5Qgp2hPNv1fDg2o5XAzHpWtcEBS5cq6F78Nv5GUxp"
    },
    "id": 1
}'
```

The following line is printed to the node's output:

```
Print from smart contract: {"fizz":{"buzz":["baz"]},"foo":"bar"}
```

That JSON is the `byteArgs` we passed in.

### Return Values

We also have a calling convention to allow returning complex values from a contract method.

The convention is that a method's literal return value (ie `return X`) is solely a success/failure indicator. If a method executes successfully, it returns 0. Otherwise, it returns some other integer. This determines the value of `invocationSuccessful` when calling `getTx`.

If the method wants to return a value, it converts it to a byte array and calls `returnValue`, which is imported from the WASM chain (see "Imported Functions" section above.) This determines the `returnValue` retrieved when calling `getTx`.

#### Example

Let's invoke contract method `get_num_bags`, which returns the number of bags that a given owner owns. This method takes one argument, the owner's ID.

```sh
curl --location --request POST 'localhost:9650/ext/bc/wasm' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "wasm.invoke",
    "params": {
        "contractID": "Enpx3HxtB6mXF3Q4deWkrWPhGACU9SQwc5FbCmyKM3uG8gS3j",
        "function": "get_num_bags",
        "senderNonce":"4",
        "senderKey":"RuuQbE5FPNwDHmkCbu9oxgR35eNFgb2eVkh6TLw7qbgqp5mn6",
        "args": [
            {
                "type": "int32",
                "value": 123
            }
        ]
    },
    "id": 1
}'
```

This returns `txID` `8UMzqASLDGo1ehWrgfnN151zZZtZh3ZDrRt4njQNRTHEmNHZv`.

To look at the return value, we call `wasm.getTx`:

```sh
{
    "jsonrpc": "2.0",
    "method": "wasm.getTx",
    "params": {
        "id": "8UMzqASLDGo1ehWrgfnN151zZZtZh3ZDrRt4njQNRTHEmNHZv"
    },
    "id": 1
}
```

The `returnValue`, when decoded, is byte array `[0 0 0 0]`. Interpreted as a big-endian integer, this is 0. (As in, the owner owns 0 bags.) 

```sh
{
    "jsonrpc": "2.0",
    "result": {
        "tx": {
            "invocationSuccessful": true,
            "returnValue": "1111XiaYg",
            "status": "Accepted",
            "tx": {
                "arguments": [
                    1
                ],
                "byteArguments": "45PJLL",
                "contractID": "8UMzqASLDGo1ehWrgfnN151zZZtZh3ZDrRt4njQNRTHEmNHZv",
                "function": "get_num_bags"
            },
            "type": "contract invocation"
        }
    },
    "id": 1
}
```

## State Management

The contract's state is persisted after every method invocation, and loaded before every method invocation.

That means that global variables, etc. in the contract that are updated internally are persisted across calls.