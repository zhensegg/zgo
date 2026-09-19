package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/zhensegg/zgo/db"
	"github.com/zhensegg/zgo/schema"
)

var ErrNotFound = errors.New("repo: not found")

type Repo[T any, ID any] struct {
	db       *db.DB
	t        *schema.Table
	cols     string
	colIdx   map[string]int
	fieldIdx []int
}

func New[T any, ID any](d *db.DB) (*Repo[T, ID], error) {
	var v T
	t, err := schema.Meta(&v)
	if err != nil {
		return nil, err
	}
	if _, err := t.PrimaryKey(); err != nil {
		return nil, err
	}

	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Pointer {
		return nil, fmt.Errorf("repo: generic type must be a struct, got %T", v)
	}
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("repo: generic type %T is not a struct", v)
	}

	byCol := map[string]int{}
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if ft.PkgPath != "" {
			continue
		}
		if c := dbColumn(ft); c != "" {
			byCol[c] = i
		}
	}

	colIdx := make(map[string]int, len(t.Columns))
	fieldIdx := make([]int, len(t.Columns))
	names := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		fi, ok := byCol[c.Name]
		if !ok {
			return nil, fmt.Errorf("repo: no model field mapped to column %q", c.Name)
		}
		colIdx[c.Name] = i
		fieldIdx[i] = fi
		names[i] = c.Name
	}

	return &Repo[T, ID]{
		db:       d,
		t:        t,
		cols:     strings.Join(names, ", "),
		colIdx:   colIdx,
		fieldIdx: fieldIdx,
	}, nil
}

func (r *Repo[T, ID]) Insert(ctx context.Context, v *T) error {
	rv := reflect.ValueOf(v).Elem()
	args := make([]any, len(r.t.Columns))
	for i := range r.t.Columns {
		args[i] = fieldVal(rv, r.fieldIdx[i])
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.t.Name, r.cols, placeholders(len(args)))
	_, err := r.db.ExecContext(ctx, q, args...)
	return err
}

func (r *Repo[T, ID]) ByID(ctx context.Context, id ID) (*T, error) {
	pk, err := r.t.PrimaryKey()
	if err != nil {
		return nil, err
	}
	q := "SELECT " + r.cols + " FROM " + r.t.Name + " WHERE " + r.soft() + pk + " = $1"
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

func (r *Repo[T, ID]) FindBy(ctx context.Context, column string, value any) (*T, error) {
	if _, ok := r.colIdx[column]; !ok {
		return nil, fmt.Errorf("repo: FindBy: unknown column %q", column)
	}
	q := "SELECT " + r.cols + " FROM " + r.t.Name + " WHERE " + r.soft() + column + " = $1"
	return r.scanOne(r.db.QueryRowContext(ctx, q, value))
}

func (r *Repo[T, ID]) All(ctx context.Context, orderBy ...string) ([]T, error) {
	q := "SELECT " + r.cols + " FROM " + r.t.Name
	if r.t.HasDeleted {
		q += " WHERE deleted_at IS NULL"
	}
	if len(orderBy) > 0 {
		q += " ORDER BY " + strings.Join(orderBy, ", ")
	}
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []T{}
	for rows.Next() {
		v, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *Repo[T, ID]) Update(ctx context.Context, v *T, fields ...string) error {
	if len(fields) == 0 {
		return errors.New("repo: Update requires at least one field")
	}
	pk, err := r.t.PrimaryKey()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v).Elem()
	sets := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)
	for i, name := range fields {
		ci, ok := r.colIdx[name]
		if !ok {
			return fmt.Errorf("repo: Update: unknown field %q", name)
		}
		if !r.t.Columns[ci].Updatable {
			return fmt.Errorf("repo: Update: field %q is not updatable", name)
		}
		sets = append(sets, name+" = $"+strconv.Itoa(i+1))
		args = append(args, fieldVal(rv, r.fieldIdx[ci]))
	}
	args = append(args, fieldVal(rv, r.fieldIdx[r.colIdx[pk]]))

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s%s = $%d",
		r.t.Name, strings.Join(sets, ", "), r.soft(), pk, len(args))
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo[T, ID]) Delete(ctx context.Context, id ID) error {
	pk, err := r.t.PrimaryKey()
	if err != nil {
		return err
	}
	var q string
	if r.t.HasDeleted {
		q = fmt.Sprintf("UPDATE %s SET deleted_at = now() WHERE %s%s = $1", r.t.Name, r.soft(), pk)
	} else {
		q = fmt.Sprintf("DELETE FROM %s WHERE %s = $1", r.t.Name, pk)
	}
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo[T, ID]) Count(ctx context.Context) (int, error) {
	q := "SELECT count(*) FROM " + r.t.Name
	if r.t.HasDeleted {
		q += " WHERE deleted_at IS NULL"
	}
	var n int
	if err := r.db.QueryRowContext(ctx, q).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *Repo[T, ID]) soft() string {
	if r.t.HasDeleted {
		return "deleted_at IS NULL AND "
	}
	return ""
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *Repo[T, ID]) scanOne(row *sql.Row) (*T, error) {
	out, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return out, err
}

func (r *Repo[T, ID]) scan(s scanner) (*T, error) {
	out := new(T)
	rv := reflect.ValueOf(out).Elem()
	dests := make([]any, len(r.t.Columns))
	for i := range r.t.Columns {
		dests[i] = rv.Field(r.fieldIdx[i]).Addr().Interface()
	}
	if err := s.Scan(dests...); err != nil {
		return nil, err
	}
	return out, nil
}

func fieldVal(rv reflect.Value, idx int) any {
	return rv.Field(idx).Interface()
}

func dbColumn(ft reflect.StructField) string {
	tag := ft.Tag.Get("db")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func placeholders(n int) string {
	ps := make([]string, n)
	for i := range ps {
		ps[i] = "$" + strconv.Itoa(i+1)
	}
	return strings.Join(ps, ", ")
}