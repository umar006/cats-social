package repository

import (
	"cats-social/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CatRepository interface {
	CreateCat(db *sql.DB, cat *domain.Cat) error
	GetAllCats(db *sql.DB, user *domain.User, queryParams url.Values) ([]domain.Cat, error)
	UpdateCat(db *sql.DB, cat *domain.Cat) error
	DeleteCat(db *sql.DB, catId uuid.UUID) error
	CheckCatExists(db *sql.DB, catId uuid.UUID, userId uuid.UUID) (bool, error)
	CheckEditableSex(db *sql.DB, cat *domain.Cat) (bool, error)
	CheckOwnerCat(ctx context.Context, tx *sql.Tx, catId uuid.UUID, userId uuid.UUID) (bool, error)
	CheckCatHasSameSex(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error)
	CheckCatHasMatched(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error)
	CheckCatFromSameOwner(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error)
	CheckBothCatExists(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error)
}

type catRepository struct{}

func NewCatRepository() CatRepository {
	return &catRepository{}
}

func (c *catRepository) CreateCat(db *sql.DB, catBody *domain.Cat) error {
	query := `INSERT INTO cats (id, created_at, name, race, sex, age_in_month, description, image_urls, owned_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
	_, err := db.Exec(query, catBody.ID, catBody.CreatedAt, catBody.Name, catBody.Race, catBody.Sex, catBody.AgeInMonth, catBody.Description, catBody.ImageUrls, catBody.OwnedById)
	if err != nil {
		return err
	}

	return nil
}

func (c *catRepository) GetAllCats(db *sql.DB, user *domain.User, queryParams url.Values) ([]domain.Cat, error) {
	query := `
		SELECT id, name, race, sex,
			age_in_month, image_urls, description,
			created_at, has_matched
		FROM cats
		WHERE deleted = false
		`

	whereClause, limitOffsetClause, args := validateGetAllCatsQueryParams(queryParams, user.Id.String())

	if len(whereClause) > 0 {
		query += "AND " + strings.Join(whereClause, " AND ")
	}
	query += "\n" + "ORDER BY created_at DESC"
	query += "\n" + strings.Join(limitOffsetClause, " ")

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cats := []domain.Cat{}
	m := pgtype.NewMap()

	for rows.Next() {
		cat := domain.Cat{}

		err = rows.Scan(&cat.ID, &cat.Name, &cat.Race, &cat.Sex, &cat.AgeInMonth, m.SQLScanner(&cat.ImageUrls), &cat.Description, &cat.CreatedAt, &cat.HasMatched)
		if err != nil {
			return nil, err
		}

		cats = append(cats, cat)
	}

	return cats, nil
}

func (c *catRepository) UpdateCat(db *sql.DB, cat *domain.Cat) error {
	query := `
		UPDATE cats
		SET name = $2,
			race = $3,
			sex = $4,
			age_in_month = $5,
			description = $6,
			image_urls = $7
		WHERE id = $1
	`
	_, err := db.Exec(query, cat.ID, cat.Name, cat.Race, cat.Sex, cat.AgeInMonth, cat.Description, cat.ImageUrls)
	if err != nil {
		return err
	}

	return nil
}

func (c *catRepository) DeleteCat(db *sql.DB, catId uuid.UUID) error {
	query := `
		UPDATE cats
		SET deleted = true
		WHERE id = $1
	`

	_, err := db.Exec(query, catId)
	if err != nil {
		return err
	}

	return nil
}

func (c *catRepository) CheckCatExists(db *sql.DB, catId uuid.UUID, userId uuid.UUID) (bool, error) {
	queryCheckCatId := `
		SELECT EXISTS (
			SELECT 1
			FROM cats
			WHERE id = $1 
				AND owned_by_id = $2
				AND deleted = false
		)
	`
	var exists bool
	row := db.QueryRow(queryCheckCatId, catId, userId)
	err := row.Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (c *catRepository) CheckEditableSex(db *sql.DB, cat *domain.Cat) (bool, error) {
	queryCheckCatId := `
		SELECT (sex != $1) as sex_diff, NOT EXISTS (
			SELECT 1
			FROM cat_matches
			WHERE user_cat_id = $2
		) as can_edit
		FROM cats
		WHERE id = $2
	`
	var sexDiff bool
	var canEdit bool
	row := db.QueryRow(queryCheckCatId, cat.Sex, cat.ID)
	err := row.Scan(&sexDiff, &canEdit)
	if err != nil {
		return false, err
	}

	return sexDiff && !canEdit, nil
}

func (c *catRepository) CheckOwnerCat(ctx context.Context, tx *sql.Tx, catId uuid.UUID, userId uuid.UUID) (bool, error) {
	queryCheckCatId := `
		SELECT EXISTS (
			SELECT 1
			FROM cats
			WHERE id = $1 
				AND owned_by_id = $2
				AND deleted = false
		)
	`
	var owner bool
	row := tx.QueryRowContext(ctx, queryCheckCatId, catId, userId)
	err := row.Scan(&owner)
	if err != nil {
		return false, err
	}

	return owner, nil
}

func (c *catRepository) CheckCatHasSameSex(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error) {
	query := `
		SELECT c1.sex = c2.sex
		FROM cats c1
			JOIN cats c2 on c2.id != c1.id
		WHERE c1.id = $1
			AND c2.id = $2
	`
	var hasSameSex bool
	err := tx.QueryRowContext(ctx, query, cat1Id, cat2Id).Scan(&hasSameSex)
	if err != nil {
		return false, err
	}

	return hasSameSex, nil
}

func (c *catRepository) CheckCatHasMatched(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error) {
	query := `
		SELECT c1.has_matched OR c2.has_matched
		FROM cats c1
			JOIN cats c2 on c2.id != c1.id
		WHERE c1.id = $1
			AND c2.id = $2
	`
	var hasMatched bool
	err := tx.QueryRowContext(ctx, query, cat1Id, cat2Id).Scan(&hasMatched)
	if err != nil {
		return false, err
	}

	return hasMatched, nil
}

func (c *catRepository) CheckCatFromSameOwner(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error) {
	query := `
		SELECT c1.owned_by_id = c2.owned_by_id
		FROM cats c1
			JOIN cats c2 on c2.id != c1.id
		WHERE c1.id = $1
			AND c2.id = $2
	`
	var sameOwner bool
	err := tx.QueryRowContext(ctx, query, cat1Id, cat2Id).Scan(&sameOwner)
	if err != nil {
		return false, err
	}

	return sameOwner, nil
}

func (c *catRepository) CheckBothCatExists(ctx context.Context, tx *sql.Tx, cat1Id uuid.UUID, cat2Id uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM cats
			WHERE id = $1
				AND deleted = false
		) AND EXISTS (
			SELECT 1
			FROM cats
			WHERE id = $2
				AND deleted = false
		) as bothExists
	`
	var bothExists bool
	err := tx.QueryRowContext(ctx, query, cat1Id, cat2Id).Scan(&bothExists)
	if err != nil {
		return false, err
	}

	return bothExists, nil
}

func validateGetAllCatsQueryParams(queryParams url.Values, userId string) ([]string, []string, []any) {
	var limitOffsetClause []string
	var whereClause []string
	var args []any

	for key, value := range queryParams {
		undefinedParam := slices.Contains(domain.CatQueryParams, key) != true
		limitOffset := key == "limit" || key == "offset"
		emptyValue := len(value[0]) < 1

		if limitOffset {
			limitOffsetClause = append(limitOffsetClause, fmt.Sprintf("%s $%d", key, len(args)+1))

			if key == "limit" && emptyValue {
				value[0] = "5"
			}
			if key == "offset" && emptyValue {
				value[0] = "0"
			}

			args = append(args, value[0])
			continue
		}

		qParamsToSkip := undefinedParam || limitOffset || emptyValue
		if qParamsToSkip {
			continue
		}

		if key == "id" {
			_, err := uuid.Parse(value[0])
			if err != nil {
				continue
			}
		}

		if key == "hasMatched" {
			key = "has_matched"
		}

		if key == "ageInMonth" {
			key = "age_in_month"

			// regex to extract operator (>,=,<) and number
			extractOperatorAndNumber := regexp.MustCompile(`([>=<])(\d+)`)
			matches := extractOperatorAndNumber.FindStringSubmatch(value[0])
			if len(matches) != 3 {
				continue
			}

			opr := matches[1]
			val := matches[2]

			whereClause = append(whereClause, fmt.Sprintf("%s %s $%d", key, opr, len(args)+1))
			args = append(args, val)

			continue
		}

		if key == "owned" {
			if value[0] != "true" && value[0] != "false" {
				continue
			}

			key = "owned_by_id"

			if value[0] == "false" {
				whereClause = append(whereClause, fmt.Sprintf("%s != $%d", key, len(args)+1))
				args = append(args, userId)
				continue
			}

			value[0] = userId
		}

		if key == "search" {
			key = "name"
		}

		whereClause = append(whereClause, fmt.Sprintf("%s = $%d", key, len(args)+1))
		args = append(args, value[0])
	}

	return whereClause, limitOffsetClause, args
}
