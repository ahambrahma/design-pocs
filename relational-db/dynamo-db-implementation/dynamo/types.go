package dynamo

type CreateTableInput struct {
	TableName  string
	Attributes []string
	Keys       KeyAttributes
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
	SortKey      *string
}

type GetItemInput struct {
	TableName            string
	PartitionKey         string
	SortKey              *string
	ProjectionExpression string
}

type QueryInput struct {
	TableName              string
	KeyConditionExpression string
	KeyValues              map[string]string
	ProjectionExpression   string
	ScanIndexForward       bool
}

type QueryOutput struct {
	Items []map[string]string
}
