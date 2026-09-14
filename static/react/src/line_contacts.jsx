/*
    line_contacts.jsx — React island for the LINE Contacts page.

    This is a self-contained React component that replaces the legacy
    jQuery script (line_contacts.js) for this one page only. It reuses the
    existing admin REST API (/api/line_contacts) and the global helpers the
    legacy app already exposes (query, flash helpers, moment), so the backend
    and every other page are completely untouched.
*/
import React, { useState, useEffect, useCallback } from "react"
import { createRoot } from "react-dom/client"

// Thin wrappers over the globals the legacy admin app already provides, so the
// island shares the same auth (Bearer api_key via query()) and UX (flashes).
const api = (path, method) => window.query(path, method, {}, false)
const flashOK = (m) => window.successFlash && window.successFlash(m)
const flashOKFade = (m) => window.successFlashFade && window.successFlashFade(m, 2)
const flashErr = (m) => window.errorFlash && window.errorFlash(m)
const fmtDate = (d) => (window.moment ? window.moment(d).format("MMMM Do YYYY, h:mm:ss a") : d)

function LineContacts() {
    const [contacts, setContacts] = useState([])
    const [loading, setLoading] = useState(true)

    const load = useCallback(() => {
        setLoading(true)
        api("/line_contacts", "GET")
            .success((cs) => {
                setContacts(cs || [])
                setLoading(false)
            })
            .error(() => {
                setLoading(false)
                flashErr("Error fetching LINE contacts")
            })
    }, [])

    useEffect(() => {
        load()
    }, [load])

    const copyId = (c) => {
        navigator.clipboard.writeText(c.line_id).then(
            () => flashOKFade("Copied userId to clipboard"),
            () => flashErr("Could not copy to clipboard")
        )
    }

    const removeContact = (c) => {
        if (!window.confirm("Remove this LINE contact?")) {
            return
        }
        api("/line_contacts/" + c.id, "DELETE")
            .success((data) => {
                flashOK(data.message)
                load()
            })
            .error((data) =>
                flashErr(data.responseJSON ? data.responseJSON.message : "Error deleting contact")
            )
    }

    return (
        <React.Fragment>
            <h1 className="page-header">
                LINE Contacts
                <button
                    type="button"
                    className="btn btn-default btn-sm pull-right"
                    onClick={load}>
                    <i className="fa fa-refresh" /> Refresh
                </button>
            </h1>
            <div className="gf-var-note">
                <div className="gf-var-ico">
                    <i className="fa fa-comment" />
                </div>
                <div>
                    <b>Auto-collected LINE userIds.</b> Point your LINE Official Account's webhook to{" "}
                    <code className="qr">https://&lt;your-public-url&gt;/line/webhook</code>. When someone
                    adds your OA as a friend or messages it, their <b>userId</b> is captured here
                    automatically &mdash; copy it into a group's <b>LINE userId</b> field to target them,
                    no manual lookup needed.
                </div>
            </div>

            {loading ? (
                <div>
                    <i className="fa fa-spinner fa-spin fa-4x" />
                </div>
            ) : contacts.length === 0 ? (
                <div className="row">
                    <div className="alert alert-info">
                        No LINE contacts yet. Once your OA webhook is set, followers will appear here.
                    </div>
                </div>
            ) : (
                <div className="row">
                    <table className="table">
                        <thead>
                            <tr>
                                <th>Display Name</th>
                                <th>LINE userId</th>
                                <th>Collected</th>
                                <th className="col-md-2"></th>
                            </tr>
                        </thead>
                        <tbody>
                            {contacts.map((c) => (
                                <tr key={c.id}>
                                    <td>{c.display_name || "—"}</td>
                                    <td>
                                        <code>{c.line_id}</code>
                                    </td>
                                    <td>{fmtDate(c.created_date)}</td>
                                    <td>
                                        <div className="pull-right">
                                            <button
                                                className="btn btn-primary"
                                                title="Copy userId"
                                                onClick={() => copyId(c)}>
                                                <i className="fa fa-copy" />
                                            </button>{" "}
                                            <button
                                                className="btn btn-danger"
                                                title="Remove"
                                                onClick={() => removeContact(c)}>
                                                <i className="fa fa-trash-o" />
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}
        </React.Fragment>
    )
}

const el = document.getElementById("react-line-contacts")
if (el) {
    createRoot(el).render(<LineContacts />)
}
