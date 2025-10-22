package dynamo

type DynamoDBTableAttributes struct {
	TableName     string
	KeyAttributes KeyAttributes
	LSIs          []LocalSecondaryIndex
}

type CreateTableInput struct {
	TableName  string
	Attributes []string
	Keys       KeyAttributes
	LSIs       []LocalSecondaryIndex
}

type LocalSecondaryIndex struct {
	IndexName string
	SortKey   string
}

type PutItemInput struct {
	TableName     string
	KeyAttributes map[string]string
	Values        map[string]string
}

type GetItemOutput struct {
	Values map[string]string
}

type KeyAttributes struct {
	PartitionKey string
	SortKey      interface{}
}

type GetItemInput struct {
	TableName            string
	PartitionKey         string
	SortKey              interface{}
	ProjectionExpression string
}

type QueryInput struct {
	TableName              string
	KeyConditionExpression string
	KeyValues              map[string]string
	ProjectionExpression   string
	ScanIndexForward       bool
	IndexName              interface{}
}

type QueryOutput struct {
	Items []map[string]string
}
