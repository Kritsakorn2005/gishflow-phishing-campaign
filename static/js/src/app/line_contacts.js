/*
    line_contacts.js
    Lists the LINE userIds auto-collected from the Official Account webhook, and
    lets the admin copy a userId (to paste into a group) or remove a contact.
    Uses the global query() helper so it works regardless of gophish.min.js
    caching.
*/
var contacts = []

function copyId(id) {
    var c = contacts.find(function (x) { return x.id === id })
    if (!c) {
        return
    }
    navigator.clipboard.writeText(c.line_id).then(function () {
        successFlashFade("Copied userId to clipboard", 2)
    }, function () {
        errorFlash("Could not copy to clipboard")
    })
}

function deleteContact(id) {
    if (!confirm("Remove this LINE contact?")) {
        return
    }
    query("/line_contacts/" + id, "DELETE", {}, false)
        .success(function (data) {
            successFlash(data.message)
            load()
        })
        .error(function (data) {
            errorFlash(data.responseJSON ? data.responseJSON.message : "Error deleting contact")
        })
}

function load() {
    $("#contactsTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    query("/line_contacts", "GET", {}, false)
        .success(function (cs) {
            contacts = cs || []
            $("#loading").hide()
            if (contacts.length > 0) {
                $("#contactsTable").show()
                var table = $("#contactsTable").DataTable({
                    destroy: true,
                    order: [], // keep the API order (newest first)
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                })
                table.clear()
                var rows = []
                $.each(contacts, function (i, c) {
                    rows.push([
                        escapeHtml(c.display_name || "—"),
                        "<code>" + escapeHtml(c.line_id) + "</code>",
                        moment(c.created_date).format("MMMM Do YYYY, h:mm:ss a"),
                        "<div class='pull-right'>\
                          <button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy userId' onclick='copyId(" + c.id + ")'><i class='fa fa-copy'></i></button>\
                          <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Remove' onclick='deleteContact(" + c.id + ")'><i class='fa fa-trash-o'></i></button>\
                        </div>"
                    ])
                })
                table.rows.add(rows).draw()
                $('[data-toggle="tooltip"]').tooltip()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching LINE contacts")
        })
}

$(document).ready(function () {
    $("#refreshBtn").on("click", load)
    load()
})
