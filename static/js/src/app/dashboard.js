var campaigns = []
var chartRegistry = []
var campaignTable = null
var currentChannel = "all"
// statuses is a helper map to point result statuses to ui classes
var statuses = {
    "Email Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "Emails Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "In progress": {
        label: "label-primary"
    },
    "Queued": {
        label: "label-info"
    },
    "Completed": {
        label: "label-success"
    },
    "Email Opened": {
        color: "#f9bf3b",
        label: "label-warning",
        icon: "fa-envelope",
        point: "ct-point-opened"
    },
    "Email Reported": {
        color: "#45d6ef",
        label: "label-warning",
        icon: "fa-bullhorne",
        point: "ct-point-reported"
    },
    "Clicked Link": {
        color: "#F39C12",
        label: "label-clicked",
        icon: "fa-mouse-pointer",
        point: "ct-point-clicked"
    },
    "Success": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Error": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Error Sending Email": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Submitted Data": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Viewed Training": {
        color: "#30d158",
        label: "label-success",
        icon: "fa-graduation-cap",
        point: "ct-point-reported"
    },
    "QR Scanned": {
        color: "#5e5ce6",
        label: "label-clicked",
        icon: "fa-qrcode",
        point: "ct-point-clicked"
    },
    "Unknown": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-question",
        point: "ct-point-error"
    },
    "Sending": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-spinner",
        point: "ct-point-sending"
    },
    "Campaign Created": {
        label: "label-success",
        icon: "fa-rocket"
    }
}

var statsMapping = {
    "sent": "Email Sent",
    "opened": "Email Opened",
    "email_reported": "Email Reported",
    "clicked": "Clicked Link",
    "submitted_data": "Submitted Data",
}

// channelBadge returns a small coloured chip labelling a campaign's delivery
// channel (Email / LINE / SMS), based on the sending profile interface type.
function channelBadge(interfaceType) {
    var map = {
        "LINE": { label: "LINE", color: "#06C755" },
        "SMS": { label: "SMS", color: "#FF9F0A" }
    }
    var c = map[interfaceType] || { label: "Email", color: "#0071E3" }
    return "<span style=\"display:inline-block;font-size:11px;font-weight:600;color:#fff;background:" +
        c.color + ";padding:1px 8px;border-radius:980px;margin-right:8px;\">" + c.label + "</span>"
}

function deleteCampaign(id) {
    var campaign = campaigns.find(function (c) {
        return c.id === id
    })
    var name = campaign ? campaign.name : "this campaign"
    if (confirm("Delete " + name + "?")) {
        api.campaignId.delete(id)
            .success(function (data) {
                successFlash(data.message)
                location.reload()
            })
    }
}

/* Renders a pie chart using the provided chartops */
function renderPieChart(chartopts) {
    return Highcharts.chart(chartopts['elemId'], {
        chart: {
            type: 'pie',
            events: {
                load: function () {
                    var chart = this,
                        rend = chart.renderer,
                        pie = chart.series[0],
                        left = chart.plotLeft + pie.center[0],
                        top = chart.plotTop + pie.center[1];
                    this.innerText = rend.text(chartopts['data'][0].count, left, top).
                    attr({
                        'text-anchor': 'middle',
                        'font-size': '16px',
                        'font-weight': 'bold',
                        'fill': chartopts['colors'][0],
                        'font-family': 'Helvetica,Arial,sans-serif'
                    }).add();
                },
                render: function () {
                    this.innerText.attr({
                        text: chartopts['data'][0].count
                    })
                }
            }
        },
        title: {
            text: chartopts['title']
        },
        plotOptions: {
            pie: {
                innerSize: '80%',
                dataLabels: {
                    enabled: false
                }
            }
        },
        credits: {
            enabled: false
        },
        tooltip: {
            formatter: function () {
                if (this.key == undefined) {
                    return false
                }
                return '<span style="color:' + this.color + '">\u25CF</span>' + this.point.name + ': <b>' + this.y + '%</b><br/>'
            }
        },
        series: [{
            data: chartopts['data'],
            colors: chartopts['colors'],
        }]
    })
}

function generateStatsPieCharts(campaigns) {
    var stats_data = []
    var stats_series_data = {}
    var total = 0

    $.each(campaigns, function (i, campaign) {
        $.each(campaign.stats, function (status, count) {
            if (status == "total") {
                total += count
                return true
            }
            if (!stats_series_data[status]) {
                stats_series_data[status] = count;
            } else {
                stats_series_data[status] += count;
            }
        })
    })
    $.each(stats_series_data, function (status, count) {
        // I don't like this, but I guess it'll have to work.
        // Turns submitted_data into Submitted Data
        if (!(status in statsMapping)) {
            return true
        }
        status_label = statsMapping[status]
        stats_data.push({
            name: status_label,
            y: Math.floor((count / total) * 100),
            count: count
        })
        stats_data.push({
            name: '',
            y: 100 - Math.floor((count / total) * 100)
        })
        var stats_chart = renderPieChart({
            elemId: status + '_chart',
            title: status_label,
            name: status,
            data: stats_data,
            colors: [statuses[status_label].color, "#dddddd"]
        })
        chartRegistry.push(stats_chart)

        stats_data = []
    });
}

function generateTimelineChart(campaigns) {
    var overview_data = []
    $.each(campaigns, function (i, campaign) {
        var campaign_date = moment.utc(campaign.created_date).local()
        // Add it to the chart data
        campaign.y = 0
        // Clicked events also contain our data submitted events
        campaign.y += campaign.stats.clicked
        campaign.y = Math.floor((campaign.y / campaign.stats.total) * 100)
        // Add the data to the overview chart
        overview_data.push({
            campaign_id: campaign.id,
            name: campaign.name,
            x: campaign_date.valueOf(),
            y: campaign.y
        })
    })
    chartRegistry.push(Highcharts.chart('overview_chart', {
        chart: {
            zoomType: 'x',
            type: 'areaspline'
        },
        title: {
            text: 'Phishing Success Overview'
        },
        xAxis: {
            type: 'datetime',
            dateTimeLabelFormats: {
                second: '%l:%M:%S',
                minute: '%l:%M',
                hour: '%l:%M',
                day: '%b %d, %Y',
                week: '%b %d, %Y',
                month: '%b %Y'
            }
        },
        yAxis: {
            min: 0,
            max: 100,
            title: {
                text: "% of Success"
            }
        },
        tooltip: {
            formatter: function () {
                return Highcharts.dateFormat('%A, %b %d %l:%M:%S %P', new Date(this.x)) +
                    '<br>' + this.point.name + '<br>% Success: <b>' + this.y + '%</b>'
            }
        },
        legend: {
            enabled: false
        },
        plotOptions: {
            series: {
                marker: {
                    enabled: true,
                    symbol: 'circle',
                    radius: 3
                },
                cursor: 'pointer',
                point: {
                    events: {
                        click: function (e) {
                            window.location.href = "/campaigns/" + this.campaign_id
                        }
                    }
                }
            }
        },
        credits: {
            enabled: false
        },
        series: [{
            data: overview_data,
            color: "#f05b4f",
            fillOpacity: 0.5
        }]
    }))
}

// clearCharts destroys all currently-rendered dashboard charts so they can be
// re-rendered when the channel filter changes.
function clearCharts() {
    chartRegistry.forEach(function (c) {
        try {
            c.destroy()
        } catch (e) {}
    })
    chartRegistry = []
}

// filterByChannel returns the campaigns matching the given channel filter.
function filterByChannel(list, channel) {
    if (channel === "all") return list
    return list.filter(function (c) {
        var it = c.interface_type
        if (channel === "email") return (!it || it === "SMTP")
        return it === channel // "LINE" or "SMS"
    })
}

// buildCampaignRow builds one DataTable row array for a campaign.
function buildCampaignRow(campaign) {
    var campaign_date = moment(campaign.created_date).format('MMMM Do YYYY, h:mm:ss a')
    var label = (statuses[campaign.status] && statuses[campaign.status].label) || "label-default";
    var quickStats
    if (moment(campaign.launch_date).isAfter(moment())) {
        quickStats = "Scheduled to start: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a') + "<br><br>Number of recipients: " + campaign.stats.total
    } else {
        quickStats = "Launch Date: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a') + "<br><br>Number of recipients: " + campaign.stats.total + "<br><br>Opened: " + campaign.stats.opened + "<br><br>Clicked: " + campaign.stats.clicked + "<br><br>Submitted: " + campaign.stats.submitted_data + "<br><br>Errors: " + campaign.stats.error + "<br><br>Reported: " + campaign.stats.email_reported
    }
    return [
        channelBadge(campaign.interface_type) + escapeHtml(campaign.name),
        campaign_date,
        campaign.stats.sent,
        campaign.stats.opened,
        campaign.stats.clicked,
        campaign.stats.submitted_data,
        campaign.stats.email_reported,
        "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"right\" data-html=\"true\" title=\"" + quickStats + "\">" + campaign.status + "</span>",
        "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" + campaign.id + "' data-toggle='tooltip' data-placement='left' title='View Results'>\
            <i class='fa fa-bar-chart'></i></a>\
            <button class='btn btn-danger' onclick='deleteCampaign(" + campaign.id + ")' data-toggle='tooltip' data-placement='left' title='Delete Campaign'>\
            <i class='fa fa-trash-o'></i></button></div>"
    ]
}

// renderDashboard (re-)renders the table and charts for the given campaign list.
function renderDashboard(list) {
    clearCharts()
    campaignTable.clear()
    var rows = []
    $.each(list, function (i, campaign) {
        rows.push(buildCampaignRow(campaign))
    })
    campaignTable.rows.add(rows).draw()
    $('[data-toggle="tooltip"]').tooltip()
    generateStatsPieCharts(list)
    generateTimelineChart(list)
}

// applyChannelFilter switches the active channel and re-renders the dashboard.
function applyChannelFilter(channel) {
    currentChannel = channel
    $("#channelFilter button").removeClass("active")
    $("#channelFilter button[data-ch='" + channel + "']").addClass("active")
    renderDashboard(filterByChannel(campaigns, channel))
}

$(document).ready(function () {
    Highcharts.setOptions({
        global: {
            useUTC: false
        }
    })
    api.campaigns.summary()
        .success(function (data) {
            $("#loading").hide()
            campaigns = data.campaigns
            if (campaigns.length > 0) {
                $("#dashboard").show()
                // Create the overview chart data
                campaignTable = $("#campaignTable").DataTable({
                    columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        },
                        {
                            className: "color-sent",
                            targets: [2]
                        },
                        {
                            className: "color-opened",
                            targets: [3]
                        },
                        {
                            className: "color-clicked",
                            targets: [4]
                        },
                        {
                            className: "color-success",
                            targets: [5]
                        },
                        {
                            className: "color-reported",
                            targets: [6]
                        }
                    ],
                    order: [
                        [1, "desc"]
                    ]
                });
                // Wire up the channel filter buttons, then render everything.
                $("#channelFilter button").on("click", function () {
                    applyChannelFilter($(this).data("ch"))
                })
                applyChannelFilter("all")
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            errorFlash("Error fetching campaigns")
        })
})
