// Code generated by ent, DO NOT EDIT.

package activity

import (
	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the activity type in the database.
	Label = "activity"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldUID holds the string denoting the uid field in the database.
	FieldUID = "uid"
	// FieldSourceUID holds the string denoting the source_uid field in the database.
	FieldSourceUID = "source_uid"
	// FieldSourceType holds the string denoting the source_type field in the database.
	FieldSourceType = "source_type"
	// FieldTitle holds the string denoting the title field in the database.
	FieldTitle = "title"
	// FieldBody holds the string denoting the body field in the database.
	FieldBody = "body"
	// FieldURL holds the string denoting the url field in the database.
	FieldURL = "url"
	// FieldImageURL holds the string denoting the image_url field in the database.
	FieldImageURL = "image_url"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldShortSummary holds the string denoting the short_summary field in the database.
	FieldShortSummary = "short_summary"
	// FieldFullSummary holds the string denoting the full_summary field in the database.
	FieldFullSummary = "full_summary"
	// FieldRawJSON holds the string denoting the raw_json field in the database.
	FieldRawJSON = "raw_json"
	// FieldEmbedding holds the string denoting the embedding field in the database.
	FieldEmbedding = "embedding"
	// Table holds the table name of the activity in the database.
	Table = "activities"
)

// Columns holds all SQL columns for activity fields.
var Columns = []string{
	FieldID,
	FieldUID,
	FieldSourceUID,
	FieldSourceType,
	FieldTitle,
	FieldBody,
	FieldURL,
	FieldImageURL,
	FieldCreatedAt,
	FieldShortSummary,
	FieldFullSummary,
	FieldRawJSON,
	FieldEmbedding,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

// OrderOption defines the ordering options for the Activity queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByUID orders the results by the uid field.
func ByUID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUID, opts...).ToFunc()
}

// BySourceUID orders the results by the source_uid field.
func BySourceUID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSourceUID, opts...).ToFunc()
}

// BySourceType orders the results by the source_type field.
func BySourceType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSourceType, opts...).ToFunc()
}

// ByTitle orders the results by the title field.
func ByTitle(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTitle, opts...).ToFunc()
}

// ByBody orders the results by the body field.
func ByBody(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBody, opts...).ToFunc()
}

// ByURL orders the results by the url field.
func ByURL(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldURL, opts...).ToFunc()
}

// ByImageURL orders the results by the image_url field.
func ByImageURL(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldImageURL, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByShortSummary orders the results by the short_summary field.
func ByShortSummary(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldShortSummary, opts...).ToFunc()
}

// ByFullSummary orders the results by the full_summary field.
func ByFullSummary(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldFullSummary, opts...).ToFunc()
}

// ByRawJSON orders the results by the raw_json field.
func ByRawJSON(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRawJSON, opts...).ToFunc()
}

// ByEmbedding orders the results by the embedding field.
func ByEmbedding(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEmbedding, opts...).ToFunc()
}
