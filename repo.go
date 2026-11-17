package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNaoEncontrada = errors.New("ocorrência não encontrada")

type Repo struct {
	pool *pgxpool.Pool
}

func NovoRepo(ctx context.Context, url string) (*Repo, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Repo{pool: pool}, nil
}

func (r *Repo) CriarTabela(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ocorrencias (
			id        SERIAL PRIMARY KEY,
			rodovia   VARCHAR(10) NOT NULL,
			tipo      VARCHAR(20) NOT NULL,
			km        INTEGER NOT NULL,
			descricao VARCHAR(200)
		)`)
	return err
}

func (r *Repo) Listar(ctx context.Context, rodovia, tipo string) ([]Ocorrencia, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, rodovia, tipo, km, descricao
		FROM ocorrencias
		WHERE ($1 = '' OR rodovia = $1) AND ($2 = '' OR tipo = $2)
		ORDER BY id`, rodovia, tipo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resultado := []Ocorrencia{}
	for rows.Next() {
		var o Ocorrencia
		if err := rows.Scan(&o.ID, &o.Rodovia, &o.Tipo, &o.Km, &o.Descricao); err != nil {
			return nil, err
		}
		resultado = append(resultado, o)
	}
	return resultado, rows.Err()
}

func (r *Repo) Buscar(ctx context.Context, id int) (Ocorrencia, error) {
	var o Ocorrencia
	err := r.pool.QueryRow(ctx,
		`SELECT id, rodovia, tipo, km, descricao FROM ocorrencias WHERE id = $1`, id,
	).Scan(&o.ID, &o.Rodovia, &o.Tipo, &o.Km, &o.Descricao)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNaoEncontrada
	}
	return o, err
}

func (r *Repo) Criar(ctx context.Context, o Ocorrencia) (Ocorrencia, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ocorrencias (rodovia, tipo, km, descricao)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, o.Rodovia, o.Tipo, o.Km, o.Descricao,
	).Scan(&o.ID)
	return o, err
}

func (r *Repo) Total(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM ocorrencias`).Scan(&n)
	return n, err
}

func (r *Repo) Deletar(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM ocorrencias WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNaoEncontrada
	}
	return nil
}

func (r *Repo) Atualizar(ctx context.Context, id int, o Ocorrencia) (Ocorrencia, error) {
	err := r.pool.QueryRow(ctx, `
		UPDATE ocorrencias
		SET rodovia = $1, tipo = $2, km = $3, descricao = $4
		WHERE id = $5
		RETURNING id`, o.Rodovia, o.Tipo, o.Km, o.Descricao, id,
	).Scan(&o.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNaoEncontrada
	}
	return o, err
}