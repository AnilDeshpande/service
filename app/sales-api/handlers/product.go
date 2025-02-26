package handlers

import (
	"context"
	"net/http"

	"github.com/ardanlabs/service/business/auth"
	"github.com/ardanlabs/service/business/data/product"
	"github.com/ardanlabs/service/foundation/messaging"
	"github.com/ardanlabs/service/foundation/web"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	kafka "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/api/global"
)

type productHandlers struct {
	db *sqlx.DB
}

func (h *productHandlers) list(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, span := global.Tracer("service").Start(ctx, "handlers.product.list")
	defer span.End()

	products, err := product.List(ctx, h.db)
	if err != nil {
		return err
	}

	return web.Respond(ctx, w, products, http.StatusOK)
}

func (h *productHandlers) retrieve(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, span := global.Tracer("service").Start(ctx, "handlers.product.retrieve")
	defer span.End()

	params := web.Params(r)
	prod, err := product.One(ctx, h.db, params["id"])
	if err != nil {
		switch err {
		case product.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case product.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return errors.Wrapf(err, "ID: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, prod, http.StatusOK)
}

func (h *productHandlers) create(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, span := global.Tracer("service").Start(ctx, "handlers.product.create")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var np product.NewProduct
	if err := web.Decode(r, &np); err != nil {
		return errors.Wrap(err, "decoding new product")
	}

	prod, err := product.Create(ctx, h.db, claims, np, v.Now)
	if err != nil {
		return errors.Wrapf(err, "creating new product: %+v", np)
	}

	//kafka producer logic
	msg := kafka.Message{
		Key:   []byte(prod.UserID), // kafka message key
		Value: []byte(prod.UserID),
	}

	if err := messaging.ProduceMessage(ctx, msg); err != nil {
		return web.NewRequestError(errors.Wrap(err, "ERROR occurs in KAKFA Producer"), http.StatusInternalServerError)
	}

	return web.Respond(ctx, w, prod, http.StatusCreated)
}

func (h *productHandlers) update(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, span := global.Tracer("service").Start(ctx, "handlers.product.update")
	defer span.End()

	claims, ok := ctx.Value(auth.Key).(auth.Claims)
	if !ok {
		return web.NewShutdownError("claims missing from context")
	}

	v, ok := ctx.Value(web.KeyValues).(*web.Values)
	if !ok {
		return web.NewShutdownError("web value missing from context")
	}

	var up product.UpdateProduct
	if err := web.Decode(r, &up); err != nil {
		return errors.Wrap(err, "")
	}

	params := web.Params(r)
	if err := product.Update(ctx, h.db, claims, params["id"], up, v.Now); err != nil {
		switch err {
		case product.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		case product.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case product.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return errors.Wrapf(err, "updating product %q: %+v", params["id"], up)
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}

func (h *productHandlers) delete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx, span := global.Tracer("service").Start(ctx, "handlers.product.delete")
	defer span.End()

	params := web.Params(r)
	if err := product.Delete(ctx, h.db, params["id"]); err != nil {
		switch err {
		case product.ErrInvalidID:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return errors.Wrapf(err, "Id: %s", params["id"])
		}
	}

	return web.Respond(ctx, w, nil, http.StatusNoContent)
}
