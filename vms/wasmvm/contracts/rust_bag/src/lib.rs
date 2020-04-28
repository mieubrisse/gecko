use std::collections::HashMap;
use lazy_static::lazy_static;
use std::sync::Mutex;
use serde_json::{Value,json};

extern "C" {
    fn print(ptr: u32, len: u32);
    fn dbPut(key_ptr: u32, key_len: u32, value_ptr: u32, value_len: u32) -> i32;
    fn dbGet(key_ptr: u32, key_len: u32, value_ptr: u32) -> i32;
    fn dbGetValueLen(key_ptr: u32, key_len: u32) -> i32;
    fn getArgs(value_ptr: u32) -> i32;
    fn getSender(ptr: u32) -> i32;
    fn returnValue(value_ptr: u32, value_len: u32) -> i32;
}

lazy_static! {
    pub static ref OWNERS: Mutex<HashMap<u32, Owner>> = Mutex::new(HashMap::new());
    pub static ref BAGS: Mutex<HashMap<u32, Bag>> = Mutex::new(HashMap::new());
}

pub enum Condition {
    New,
    Good,
    Bad,
    Destroyed
}

#[derive(Debug)]
pub struct Owner {
    pub id: u32,
    pub bags: Vec<u32> // Each element is the ID of a bag this owner owns
}

pub struct Bag {
    pub id: u32,
    pub price: u32,
    pub owner_id: u32,
    pub num_transfers: u32,
    pub condition: Condition
}

// Create a new owner
// Returns 0 on success
// Returns 1 otherwise
#[no_mangle]
pub extern fn create_owner() -> i32 {
    unsafe {
        // Get args as raw bytes
        let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        args.set_len(args_len as usize);
        // Parse args to json
        let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
        let json : serde_json::Value;
        match args_json {
            Ok(some) => json = some,
            Err(_) => return -1,
        }
        // Get "owner_id" field from json
        let owner_id_value = &json["owner_id"];
        let owner_id: u32;
        match owner_id_value {
            Value::Number(id) => owner_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }
        // Create the owner
        let mut owners = OWNERS.lock().unwrap();
        match owners.get(&owner_id) {
            Some(_) => -1, // This owner already exists
            None => {
                owners.insert(owner_id, Owner{
                    id: owner_id,
                    bags: Vec::new()
                });
                0
            }
        }
    }
}

// Create a new bag
// Precondition: There is an owner with the specified ID
// Returns 0 on success
// Returns -1 otherwise
#[no_mangle]
pub extern fn create_bag() -> i32 {
    unsafe {
        // Get args as raw bytes
        let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        args.set_len(args_len as usize);

        // Parse args to json
        let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
        let json : serde_json::Value;
        match args_json {
            Ok(some) => json = some,
            Err(_) => return -1,
        }

        // Get "owner_id" field from json
        let owner_id: u32;
        match &json["owner_id"] {
            Value::Number(id) => owner_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

         // Get "bag_id" field from json
         let bag_id: u32;
         match &json["bag_id"] {
             Value::Number(id) => bag_id = id.as_u64().unwrap() as u32,
             _ => return -1,
         }
         
         // Get "price" field
         let price: u32;
         match &json["price"] {
             Value::Number(id) => price = id.as_u64().unwrap() as u32,
             _ => return -1,
         }

         // Get "condition" field
         let condition: Condition;
         match &json["condition"] {
                    Value::String(s) => match s.to_lowercase().as_str() {
                        "new" => condition = Condition::New,
                        "good" => condition = Condition::Good,
                        "bad" => condition = Condition::Bad,
                        "destroyed" => condition = Condition::Destroyed,
                        _ => return -1
                    }
                    _ => return -1
        }

        // Check that the bag doesn't exist and the owner does 
        let mut bags = BAGS.lock().unwrap();
        if let Some(_) = bags.get(&bag_id) {
            return 1 // failure
        }

        let owners = &mut OWNERS.lock().unwrap();
        if let None = owners.get(&owner_id) {
            return 1
        }

        // Update bag list
        bags.insert(bag_id, Bag{
                id: bag_id,
                owner_id: owner_id,
                num_transfers: 0,
                price: price,
                condition: condition,
            }
        );
        
        // Update the owner
        owners.get_mut(&owner_id).unwrap().bags.push(bag_id);
        0 //success
    }
}

// Update the specified bag's price
// Returns 0 on success
// Returns 1 if the bag doesn't exist
#[no_mangle]
pub extern fn update_bag_price() -> i32 {
    unsafe {
        // Get args as raw bytes
        let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        args.set_len(args_len as usize);

        // Parse args to json
        let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
        let json : serde_json::Value;
        match args_json {
            Ok(some) => json = some,
            Err(_) => return -1,
        }

        // Get "bag_id" field from json
        let bag_id: u32;
        match &json["bag_id"] {
            Value::Number(id) => bag_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

        // Get "price" field from json
        let price: u32;
        match &json["price"] {
            Value::Number(id) => price = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

        let bags = &mut BAGS.lock().unwrap();
        if let Some(bag) = bags.get_mut(&bag_id) {
            bag.price=price;
            0
        } else {
            1
        }
    }
}

// Return the owner with the specified ID
// Returns 1 if the owner doesn't exist
#[no_mangle]
pub extern fn get_owner() -> i32 {
    unsafe {
        // Get args as raw bytes
        let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        args.set_len(args_len as usize);

        // Parse args to json
        let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
        let json : serde_json::Value;
        match args_json {
            Ok(some) => json = some,
            Err(_) => return -1,
        }

        // Get "owner_id" field from json
        let owner_id: u32;
        match &json["owner_id"] {
            Value::Number(id) => owner_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

        if let Some(owner) = OWNERS.lock().unwrap().get(&owner_id) {
            let response = json!({
                "ownerID": owner.id,
                "bags": owner.bags
            });
            let response = serde_json::to_vec(&response);
            match response {
                Ok(json) => {
                    let ptr = json.as_ptr();
                    returnValue(ptr as u32, json.len() as u32)
                },
                Err(_) => 1
            }
        } else {
            1
        }
    }
}

// Return the ID of the owner of the specified bag
// Returns 1 if the bag doesn't exist
#[no_mangle]
pub extern fn get_bag() -> i32 {
    unsafe {
         // Get args as raw bytes
         let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
         let pointer = args.as_ptr() as u32;
         let args_len = getArgs(pointer);
         if args_len == -1 {
             return -1;
         }
         args.set_len(args_len as usize);
 
         // Parse args to json
         let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
         let json : serde_json::Value;
         match args_json {
             Ok(some) => json = some,
             Err(_) => return -1,
         }
 
         // Get "bag_id" field from json
         let bag_id: u32;
         match &json["bag_id"] {
             Value::Number(id) => bag_id = id.as_u64().unwrap() as u32,
             _ => return -1,
         }

        if let Some(bag) = BAGS.lock().unwrap().get(&bag_id) {
            let condition = match bag.condition {
                Condition::New => "new",
                Condition::Good => "good",
                Condition::Bad => "bad",
                Condition::Destroyed => "destroyed"
            };
            let response = json!({
                "ID": bag.id,
                "price": bag.price,
                "ownerID": bag.owner_id,
                "condition": condition
            });
            let response = serde_json::to_vec(&response);
            match response {
                Ok(json) => {
                    let ptr = json.as_ptr();
                    returnValue(ptr as u32, json.len() as u32)
                },
                Err(_) => 1
            }
        } else {
            1
        }
    }
}

// Transfer a bag to a new owner
// Returns 1 if the bag or new owner don't exist
#[no_mangle]
pub extern fn transfer_bag() -> i32 {
    unsafe {
        // Get args as raw bytes
        let args: &mut std::vec::Vec<u8> = &mut Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        args.set_len(args_len as usize);

        // Parse args to json
        let args_json: std::result::Result<serde_json::Value, serde_json::error::Error> = serde_json::from_slice(&args[..args_len as usize]);
        let json : serde_json::Value;
        match args_json {
            Ok(some) => json = some,
            Err(_) => return -1,
        }

        // Get "bag_id" field from json
        let bag_id: u32;
        match &json["bag_id"] {
            Value::Number(id) => bag_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

        // Get "owner_id" field from json
        let owner_id: u32;
        match &json["owner_id"] {
            Value::Number(id) => owner_id = id.as_u64().unwrap() as u32,
            _ => return -1,
        }

        // Check that the bag and new owner exist 
        let mut bags = BAGS.lock().unwrap();
        let bag: &mut Bag;
        if let Some(_bag) = bags.get_mut(&bag_id) {
        bag = _bag;
        } else {
            return 1 // bag doesn't exist
        }

        let owners = &mut OWNERS.lock().unwrap();
        let new_owner: &mut Owner;
        if let Some(owner) = owners.get_mut(&owner_id) {
            new_owner = owner;
        } else {
            return 1 // new owner doesn't exist
        }

        // Update the new owner
        new_owner.bags.push(bag.id);

        // Update the bag's current owner
        // get the index of the bag in the current owner's bag list
        let current_owner = owners.get_mut(&bag.owner_id).unwrap();
        let index = current_owner.bags.iter().position(|&r| r == bag_id).unwrap();
        owners.get_mut(&bag.owner_id).unwrap().bags.remove(index);

        // Update the bag
        bag.owner_id = owner_id;    

        0 // success
    }
}

// Prints "Hello, world!"
#[no_mangle]
pub extern fn say_hello() {
    let ptr = b"Hello, world!".as_ptr();
    unsafe {print(ptr as u32, 13);}
}

// Put KV pair "hello" -> "world" in the contract's DB
#[no_mangle]
pub extern fn put_hello() {
    let key_ptr = b"hello".as_ptr();
    let value_ptr = b"world".as_ptr();
    unsafe {dbPut(key_ptr as u32, 5, value_ptr as u32, 5);}
}

// print byte arguments to this method
#[no_mangle]
pub extern fn print_byte_args() -> i32 {
    unsafe {
        let args: std::vec::Vec<u8> = Vec::with_capacity(1024 as usize);
        let pointer = args.as_ptr() as u32;
        let args_len = getArgs(pointer);
        if args_len == -1 {
            return -1;
        }
        print(pointer, args_len as u32);
        0
    }
}

// print the sender that invoked this method
#[no_mangle]
pub extern fn print_sender() -> i32 {
    unsafe {
        let sender: std::vec::Vec<u8> = Vec::with_capacity(20 as usize);
        let pointer = sender.as_ptr() as u32;
        let sender_len = getSender(pointer);
        if sender_len == -1 {
            return -1;
        }
        print(pointer, 20);
        0
    }
}