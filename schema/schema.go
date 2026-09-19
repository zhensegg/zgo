package schema

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zhensegg/zgo/db"
)

type Namer interface {
	TableName() string
}

type Column struct {
	Name      string
	Typ       string
	PK        bool
	Unique    bool
	Index     bool
	NotNull   bool
	Default   string
	Updatable bool
}

type Table struct {
	Name       string
	Columns    []Column
	HasDeleted bool
}

func (t *Table) PrimaryKey() (string, error) {
	var pk string
	for _, c := range t.Columns {
		if c.PK {
			if pk != "" {
				return "", fmt.Errorf("schema: %s: multiple primary keys", t.Name)
			}
			pk = c.Name
		}
	}
	if pk == "" {
		return "", fmt.Errorf("schema: %s: no primary key", t.Name)
	}
	return pk, nil
}

func Create(ctx context.Context, d *db.DB, models ...any) error {
	for _, m := range models {
		t, err := Meta(m)
		if err != nil {
			return err
		}
		if err := createTable(ctx, d, t); err != nil {
			return err
		}
	}
	return nil
}

func Meta(m any) (*Table, error) {
	t, err := tableOf(m)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func createTable(ctx context.Context, d *db.DB, t *Table) error {
	var b strings.Builder
	b.WriteString("CREATE TABLE IF NOT EXISTS " + t.Name + " (\n")
	for i, c := range t.Columns {
		b.WriteString("\t" + c.Name + " " + c.Typ)
		if c.PK {
			b.WriteString(" PRIMARY KEY")
		} else if c.NotNull {
			b.WriteString(" NOT NULL")
		}
		if c.Default != "" {
			b.WriteString(" DEFAULT " + c.Default)
		}
		if i < len(t.Columns)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(")")

	if _, err := d.ExecContext(ctx, b.String()); err != nil {
		return fmt.Errorf("schema: create table %s: %w", t.Name, err)
	}

	for _, c := range t.Columns {
		if !c.Unique && !c.Index {
			continue
		}
		kind, suffix := "INDEX", "idx"
		if c.Unique {
			kind, suffix = "UNIQUE INDEX", "key"
		}
		partial := ""
		if c.Unique && t.HasDeleted {
			partial = " WHERE deleted_at IS NULL"
		}
		q := fmt.Sprintf("CREATE %s IF NOT EXISTS %s_%s_%s ON %s (%s)%s",
			kind, t.Name, c.Name, suffix, t.Name, c.Name, partial)
		if _, err := d.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("schema: index %s.%s: %w", t.Name, c.Name, err)
		}
	}
	return nil
}

func tableOf(m any) (Table, error) {
	rt := reflect.TypeOf(m)
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return Table{}, fmt.Errorf("schema: %T is not a struct", m)
	}
	if rt.NumField() == 0 {
		return Table{}, fmt.Errorf("schema: %s has no fields", rt.Name())
	}

	t := Table{Name: tableName(m, rt)}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.PkgPath != "" {
			continue // unexported
		}
		if f.Tag.Get("db") == "-" {
			continue
		}
		c, err := columnOf(f)
		if err != nil {
			return Table{}, fmt.Errorf("schema: %s.%s: %w", rt.Name(), f.Name, err)
		}
		t.Columns = append(t.Columns, c)
		if c.Name == "deleted_at" {
			t.HasDeleted = true
		}
	}
	if len(t.Columns) == 0 {
		return Table{}, fmt.Errorf("schema: %s has no mapped columns", rt.Name())
	}
	return t, nil
}

func tableName(m any, rt reflect.Type) string {
	if n, ok := m.(Namer); ok {
		return n.TableName()
	}
	return strings.ToLower(rt.Name()) + "s"
}

func columnOf(f reflect.StructField) (Column, error) {
	tag := f.Tag.Get("db")
	if tag == "" {
		return Column{}, fmt.Errorf("missing db tag")
	}
	parts := strings.Split(tag, ",")
	c := Column{Name: parts[0]}

	typ, nullable, err := sqlType(f.Type)
	if err != nil {
		return Column{}, err
	}
	c.Typ = typ
	c.NotNull = !nullable

	for _, opt := range parts[1:] {
		switch {
		case opt == "pk":
			c.PK = true
		case opt == "unique":
			c.Unique = true
		case opt == "index":
			c.Index = true
		case opt == "updatable":
			c.Updatable = true
		case strings.HasPrefix(opt, "default="):
			c.Default = strings.TrimPrefix(opt, "default=")
		default:
			return Column{}, fmt.Errorf("unknown db option %q", opt)
		}
	}
	return c, nil
}

var (
	uuidType  = reflect.TypeOf(uuid.UUID{})
	timeType  = reflect.TypeOf(time.Time{})
	bytesType = reflect.TypeOf([]byte(nil))
)

func sqlType(t reflect.Type) (string, bool, error) {
	nullable := false
	if t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}

	if t == timeType {
		return "timestamptz", nullable, nil
	}
	if t == uuidType {
		return "uuid", nullable, nil
	}
	if t == bytesType {
		return "bytea", nullable, nil
	}

	switch t.Kind() {
	case reflect.String:
		return "text", nullable, nil
	case reflect.Bool:
		return "boolean", nullable, nil
	case reflect.Int, reflect.Int32:
		return "integer", nullable, nil
	case reflect.Int64:
		return "bigint", nullable, nil
	case reflect.Float64:
		return "double precision", nullable, nil
	case reflect.Float32:
		return "real", nullable, nil
	case reflect.Slice:
		return "text[]", nullable, nil
	default:
		return "", false, fmt.Errorf("unsupported type %s", t)
	}
}