/*
    risk.js
    Renders the GishFlow risk-scoring dashboard: per-recipient and per-group
    (department) susceptibility, computed on demand from campaign results.
*/

function levelPill(level) {
    var cls = "gf-risk-low"
    if (level === "High") cls = "gf-risk-high"
    else if (level === "Medium") cls = "gf-risk-med"
    else if (level === "No data") cls = "gf-risk-none"
    return "<span class='gf-risk-pill " + cls + "'>" + level + "</span>"
}

function scoreCell(score, level) {
    var color = "var(--gf-green)"
    if (level === "High") color = "var(--gf-red)"
    else if (level === "Medium") color = "var(--gf-amber)"
    else if (level === "No data") { color = "var(--gf-text-3)"; }
    return "<div class='gf-score'><div class='gf-score-track'><div class='gf-score-fill' style='width:" + score + "%;background:" + color + "'></div></div><b>" + score + "</b></div>"
}

function donut(low, med, high) {
    var total = low + med + high
    if (total === 0) total = 1
    var C = 2 * Math.PI * 60
    var segs = [
        [low, "var(--gf-green)"],
        [med, "var(--gf-amber)"],
        [high, "var(--gf-red)"]
    ]
    var offset = 0
    var circles = ""
    segs.forEach(function (s) {
        var frac = s[0] / total
        var len = C * frac
        circles += "<circle cx='75' cy='75' r='60' fill='none' stroke='" + s[1] + "' stroke-width='18' stroke-dasharray='" + len + " " + (C - len) + "' stroke-dashoffset='" + (-offset) + "'></circle>"
        offset += len
    })
    return "<svg width='150' height='150' viewBox='0 0 150 150' style='transform:rotate(-90deg)'>" +
        "<circle cx='75' cy='75' r='60' fill='none' stroke='var(--gf-surface-3)' stroke-width='18'></circle>" +
        circles + "</svg>"
}

function render(users, groups) {
    $("#riskLoading").hide()
    if (!users || users.length === 0) {
        $("#emptyRisk").show()
        return
    }
    $("#riskContent").show()

    var low = 0, med = 0, high = 0, sum = 0
    users.forEach(function (u) {
        sum += u.score
        if (u.level === "High") high++
        else if (u.level === "Medium") med++
        else low++
    })
    var avg = Math.round(sum / users.length)

    // KPIs
    $("#kpiAvg").text(avg)
    $("#kpiAvgLevel").html(levelPill(avg >= 61 ? "High" : (avg >= 31 ? "Medium" : "Low")))
    $("#kpiHigh").text(high)
    $("#kpiParticipants").text(users.length)

    // Distribution donut + legend
    $("#donut").html(donut(low, med, high))
    $("#legend").html(
        "<div class='gf-li'><i style='background:var(--gf-green)'></i>Low<span>" + low + "</span></div>" +
        "<div class='gf-li'><i style='background:var(--gf-amber)'></i>Medium<span>" + med + "</span></div>" +
        "<div class='gf-li'><i style='background:var(--gf-red)'></i>High<span>" + high + "</span></div>"
    )

    // Users table
    var rows = ""
    users.forEach(function (u) {
        var name = escapeHtml(((u.first_name || "") + " " + (u.last_name || "")).trim() || u.email)
        rows += "<tr><td><span class='name'>" + name + "</span></td>" +
            "<td class='muted-cell'>" + escapeHtml(u.email) + "</td>" +
            "<td>" + escapeHtml(u.position || "") + "</td>" +
            "<td class='num'>" + u.campaigns + "</td>" +
            "<td class='num'>" + u.submitted + "</td>" +
            "<td class='num'>" + u.clicked + "</td>" +
            "<td class='num'>" + u.reported + "</td>" +
            "<td class='num'>" + u.trained + "</td>" +
            "<td>" + scoreCell(u.score, u.level) + "</td>" +
            "<td>" + levelPill(u.level) + "</td></tr>"
    })
    $("#userRows").html(rows)

    // Groups table
    var grows = ""
    groups.forEach(function (g) {
        grows += "<tr><td><span class='name'>" + escapeHtml(g.name) + "</span></td>" +
            "<td class='num'>" + g.members + "</td>" +
            "<td class='num'>" + g.participants + "</td>" +
            "<td>" + (g.participants > 0 ? scoreCell(g.score, g.level) : "<span class='muted-cell'>&mdash;</span>") + "</td>" +
            "<td>" + levelPill(g.level) + "</td></tr>"
    })
    $("#groupRows").html(grows)
}

$(document).ready(function () {
    api.risk.users()
        .success(function (users) {
            api.risk.groups()
                .success(function (groups) {
                    render(users, groups)
                })
                .error(function () {
                    render(users, [])
                })
        })
        .error(function () {
            $("#riskLoading").hide()
            errorFlash("Error computing risk scores")
        })
})
