package main

import "dynamo-db-implementation/dynamo"

func main() {
	ddb := dynamo.NewDynamoDB()
	userId := "user_id"
	orderId := "order_id"
	ddb.CreateTable(dynamo.CreateTableInput{
		TableName: "users",
		Attributes: []string{
			"org_id",
			"user_id",
			"name",
			"age"},
		Keys: dynamo.KeyAttributes{
			PartitionKey: "org_id",
			SortKey:      &userId,
		},
	})

	ddb.CreateTable(dynamo.CreateTableInput{
		TableName: "orders",
		Attributes: []string{
			"order_id",
			"user_id",
			"amount",
			"status"},
		Keys: dynamo.KeyAttributes{
			PartitionKey: userId,
			SortKey:      &orderId,
		},
	})

	ddb.PutItem(dynamo.PutItemInput{
		TableName: "users",
		KeyAttributes: map[string]string{
			"org_id":  "org#1",
			"user_id": "user#1",
		},
		Values: map[string]string{
			"name": "Shubham Sharma",
			"age":  "28",
		},
	})

	ddb.PutItem(dynamo.PutItemInput{
		TableName: "users",
		KeyAttributes: map[string]string{
			"org_id":  "org#1",
			"user_id": "user#2",
		},
		Values: map[string]string{
			"name": "John Doe",
			"age":  "26",
		},
	})

	ddb.PutItem(dynamo.PutItemInput{
		TableName: "orders",
		KeyAttributes: map[string]string{
			"order_id": "order#1",
			"user_id":  "user#1",
		},
		Values: map[string]string{
			"amount": "250",
			"status": "pending",
		},
	})

	ddb.PutItem(dynamo.PutItemInput{
		TableName: "orders",
		KeyAttributes: map[string]string{
			"order_id": "order#2",
			"user_id":  "user#1",
		},
		Values: map[string]string{
			"amount": "450",
			"status": "completed",
		},
	})

	user1 := "user#1"

	ddb.GetItem(dynamo.GetItemInput{
		TableName:            "users",
		PartitionKey:         "org#1",
		SortKey:              &user1,
		ProjectionExpression: "org_id,user_id,name,age",
	})

	ddb.Query(dynamo.QueryInput{
		TableName:              "orders",
		KeyConditionExpression: "user_id = ?",
		KeyValues: map[string]string{
			"user_id": "user#1",
		},
		ProjectionExpression: "user_id,order_id,amount,status",
		ScanIndexForward:     false,
	})
}
