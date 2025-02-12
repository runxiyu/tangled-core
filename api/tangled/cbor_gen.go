// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package tangled

import (
	"fmt"
	"io"
	"math"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

func (t *PublicKey) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.Key (string) (string)
	if len("key") > 1000000 {
		return xerrors.Errorf("Value in field \"key\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("key"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("key")); err != nil {
		return err
	}

	if len(t.Key) > 1000000 {
		return xerrors.Errorf("Value in field t.Key was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Key))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Key)); err != nil {
		return err
	}

	// t.Name (string) (string)
	if len("name") > 1000000 {
		return xerrors.Errorf("Value in field \"name\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("name"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("name")); err != nil {
		return err
	}

	if len(t.Name) > 1000000 {
		return xerrors.Errorf("Value in field t.Name was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Name))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Name)); err != nil {
		return err
	}

	// t.LexiconTypeID (string) (string)
	if len("$type") > 1000000 {
		return xerrors.Errorf("Value in field \"$type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("$type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("$type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sh.tangled.publicKey"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("sh.tangled.publicKey")); err != nil {
		return err
	}

	// t.Created (string) (string)
	if len("created") > 1000000 {
		return xerrors.Errorf("Value in field \"created\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("created"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("created")); err != nil {
		return err
	}

	if len(t.Created) > 1000000 {
		return xerrors.Errorf("Value in field t.Created was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Created))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Created)); err != nil {
		return err
	}
	return nil
}

func (t *PublicKey) UnmarshalCBOR(r io.Reader) (err error) {
	*t = PublicKey{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("PublicKey: map struct too large (%d)", extra)
	}

	n := extra

	nameBuf := make([]byte, 7)
	for i := uint64(0); i < n; i++ {
		nameLen, ok, err := cbg.ReadFullStringIntoBuf(cr, nameBuf, 1000000)
		if err != nil {
			return err
		}

		if !ok {
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(cr, func(cid.Cid) {}); err != nil {
				return err
			}
			continue
		}

		switch string(nameBuf[:nameLen]) {
		// t.Key (string) (string)
		case "key":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Key = string(sval)
			}
			// t.Name (string) (string)
		case "name":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Name = string(sval)
			}
			// t.LexiconTypeID (string) (string)
		case "$type":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.LexiconTypeID = string(sval)
			}
			// t.Created (string) (string)
		case "created":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Created = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(r, func(cid.Cid) {}); err != nil {
				return err
			}
		}
	}

	return nil
}
func (t *KnotMember) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)
	fieldCount := 4

	if t.AddedAt == nil {
		fieldCount--
	}

	if _, err := cw.Write(cbg.CborEncodeMajorType(cbg.MajMap, uint64(fieldCount))); err != nil {
		return err
	}

	// t.LexiconTypeID (string) (string)
	if len("$type") > 1000000 {
		return xerrors.Errorf("Value in field \"$type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("$type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("$type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sh.tangled.knot.member"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("sh.tangled.knot.member")); err != nil {
		return err
	}

	// t.Domain (string) (string)
	if len("domain") > 1000000 {
		return xerrors.Errorf("Value in field \"domain\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("domain"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("domain")); err != nil {
		return err
	}

	if len(t.Domain) > 1000000 {
		return xerrors.Errorf("Value in field t.Domain was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Domain))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Domain)); err != nil {
		return err
	}

	// t.Member (string) (string)
	if len("member") > 1000000 {
		return xerrors.Errorf("Value in field \"member\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("member"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("member")); err != nil {
		return err
	}

	if len(t.Member) > 1000000 {
		return xerrors.Errorf("Value in field t.Member was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Member))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Member)); err != nil {
		return err
	}

	// t.AddedAt (string) (string)
	if t.AddedAt != nil {

		if len("addedAt") > 1000000 {
			return xerrors.Errorf("Value in field \"addedAt\" was too long")
		}

		if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("addedAt"))); err != nil {
			return err
		}
		if _, err := cw.WriteString(string("addedAt")); err != nil {
			return err
		}

		if t.AddedAt == nil {
			if _, err := cw.Write(cbg.CborNull); err != nil {
				return err
			}
		} else {
			if len(*t.AddedAt) > 1000000 {
				return xerrors.Errorf("Value in field t.AddedAt was too long")
			}

			if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(*t.AddedAt))); err != nil {
				return err
			}
			if _, err := cw.WriteString(string(*t.AddedAt)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *KnotMember) UnmarshalCBOR(r io.Reader) (err error) {
	*t = KnotMember{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("KnotMember: map struct too large (%d)", extra)
	}

	n := extra

	nameBuf := make([]byte, 7)
	for i := uint64(0); i < n; i++ {
		nameLen, ok, err := cbg.ReadFullStringIntoBuf(cr, nameBuf, 1000000)
		if err != nil {
			return err
		}

		if !ok {
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(cr, func(cid.Cid) {}); err != nil {
				return err
			}
			continue
		}

		switch string(nameBuf[:nameLen]) {
		// t.LexiconTypeID (string) (string)
		case "$type":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.LexiconTypeID = string(sval)
			}
			// t.Domain (string) (string)
		case "domain":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Domain = string(sval)
			}
			// t.Member (string) (string)
		case "member":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Member = string(sval)
			}
			// t.AddedAt (string) (string)
		case "addedAt":

			{
				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					sval, err := cbg.ReadStringWithMax(cr, 1000000)
					if err != nil {
						return err
					}

					t.AddedAt = (*string)(&sval)
				}
			}

		default:
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(r, func(cid.Cid) {}); err != nil {
				return err
			}
		}
	}

	return nil
}
func (t *GraphFollow) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{163}); err != nil {
		return err
	}

	// t.LexiconTypeID (string) (string)
	if len("$type") > 1000000 {
		return xerrors.Errorf("Value in field \"$type\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("$type"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("$type")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sh.tangled.graph.follow"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("sh.tangled.graph.follow")); err != nil {
		return err
	}

	// t.Subject (string) (string)
	if len("subject") > 1000000 {
		return xerrors.Errorf("Value in field \"subject\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("subject"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("subject")); err != nil {
		return err
	}

	if len(t.Subject) > 1000000 {
		return xerrors.Errorf("Value in field t.Subject was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Subject))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.Subject)); err != nil {
		return err
	}

	// t.CreatedAt (string) (string)
	if len("createdAt") > 1000000 {
		return xerrors.Errorf("Value in field \"createdAt\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("createdAt"))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string("createdAt")); err != nil {
		return err
	}

	if len(t.CreatedAt) > 1000000 {
		return xerrors.Errorf("Value in field t.CreatedAt was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.CreatedAt))); err != nil {
		return err
	}
	if _, err := cw.WriteString(string(t.CreatedAt)); err != nil {
		return err
	}
	return nil
}

func (t *GraphFollow) UnmarshalCBOR(r io.Reader) (err error) {
	*t = GraphFollow{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("GraphFollow: map struct too large (%d)", extra)
	}

	n := extra

	nameBuf := make([]byte, 9)
	for i := uint64(0); i < n; i++ {
		nameLen, ok, err := cbg.ReadFullStringIntoBuf(cr, nameBuf, 1000000)
		if err != nil {
			return err
		}

		if !ok {
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(cr, func(cid.Cid) {}); err != nil {
				return err
			}
			continue
		}

		switch string(nameBuf[:nameLen]) {
		// t.LexiconTypeID (string) (string)
		case "$type":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.LexiconTypeID = string(sval)
			}
			// t.Subject (string) (string)
		case "subject":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.Subject = string(sval)
			}
			// t.CreatedAt (string) (string)
		case "createdAt":

			{
				sval, err := cbg.ReadStringWithMax(cr, 1000000)
				if err != nil {
					return err
				}

				t.CreatedAt = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			if err := cbg.ScanForLinks(r, func(cid.Cid) {}); err != nil {
				return err
			}
		}
	}

	return nil
}
