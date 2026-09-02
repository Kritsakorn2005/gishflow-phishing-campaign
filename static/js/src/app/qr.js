/*
    qr.js
    Standalone QR code generator — builds a QR for any URL/text via the
    /api/qr endpoint, for physical quishing tests (posters, printed codes).
*/

function generateQR() {
    var text = $("#qr_text").val().trim()
    if (!text) {
        errorFlash("Please enter a URL or text")
        return
    }
    var size = $("#qr_size").val() || "512"
    var src = "/api/qr?api_key=" + encodeURIComponent(user.api_key) +
        "&size=" + encodeURIComponent(size) +
        "&text=" + encodeURIComponent(text)
    $("#qr_image").attr("src", src)
    $("#qr_download").attr("href", src)
    $("#qr_empty").hide()
    $("#qr_result").show()
}

$(document).ready(function () {
    $("#qr_generate").click(generateQR)
    $("#qr_text").keypress(function (e) {
        if (e.which === 13) {
            generateQR()
        }
    })
})
