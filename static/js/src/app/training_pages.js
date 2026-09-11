/*
    training_pages.js
    Handles the creation, editing, and deletion of GishFlow training
    (teachable-moment) pages.
*/
var pages = []
var quizQid = 0

// addChoice appends one answer choice (radio + text) to a question block.
function addChoice($q, text, correct) {
    var name = "correct-" + $q.data("qid")
    var $c = $('<div class="gfq-choice" style="display:flex;align-items:center;margin:4px 0;">' +
        '<input type="radio" name="' + name + '" class="gfq-correct" title="Mark as correct answer"' + (correct ? ' checked' : '') + '>' +
        '<input type="text" class="form-control gfq-choice-text" placeholder="Answer choice" style="margin:0 8px;">' +
        '<button type="button" class="btn btn-link btn-xs gfq-del-choice" title="Remove choice">&times;</button>' +
        '</div>')
    $c.find(".gfq-choice-text").val(text || "")
    $q.find(".gfq-choices").append($c)
}

// addQuestion appends a question block. Pass a {q, choices, answer} object to
// pre-fill it (editing), or nothing for a blank question with two choices.
function addQuestion(q) {
    quizQid++
    var $q = $('<div class="gf-quiz-q panel panel-default" data-qid="' + quizQid + '" style="padding:12px;margin-bottom:10px;">' +
        '<div class="form-group" style="margin-bottom:8px;">' +
        '<input type="text" class="form-control gfq-text" placeholder="Question"></div>' +
        '<div class="gfq-choices"></div>' +
        '<button type="button" class="btn btn-default btn-xs gfq-add-choice" style="margin-top:6px;"><i class="fa fa-plus"></i> Add choice</button>' +
        '<button type="button" class="btn btn-danger btn-xs gfq-del-q pull-right" style="margin-top:6px;">Remove question</button>' +
        '</div>')
    $("#quizBuilder").append($q)
    if (q) {
        $q.find(".gfq-text").val(q.q)
        $.each(q.choices, function (j, c) {
            addChoice($q, c, j === q.answer)
        })
    } else {
        addChoice($q, "", true)
        addChoice($q, "", false)
    }
}

// collectQuiz reads the builder DOM into an array of quiz questions. Questions
// without text or with fewer than two choices are skipped.
function collectQuiz() {
    var out = []
    $("#quizBuilder .gf-quiz-q").each(function () {
        var qtext = $(this).find(".gfq-text").val().trim()
        var choices = []
        var answer = 0
        $(this).find(".gfq-choice").each(function (j) {
            choices.push($(this).find(".gfq-choice-text").val())
            if ($(this).find(".gfq-correct").is(":checked")) {
                answer = j
            }
        })
        if (qtext && choices.length >= 2) {
            out.push({ q: qtext, choices: choices, answer: answer })
        }
    })
    return out
}

// renderQuiz clears and repopulates the builder from a stored quiz JSON string.
function renderQuiz(quizStr) {
    $("#quizBuilder").empty()
    quizQid = 0
    if (!quizStr) {
        return
    }
    try {
        var arr = JSON.parse(quizStr)
        $.each(arr, function (i, q) {
            addQuestion(q)
        })
    } catch (e) { /* malformed quiz, start empty */ }
}

// aiGenerateQuiz asks the AI backend for quiz questions based on the current
// training content and the chosen options, then appends them to the builder.
function aiGenerateQuiz() {
    var editor = CKEDITOR.instances["html_editor"]
    var content = editor ? editor.getData() : $("#html_editor").val()
    var topic = $("#ai_quiz_topic").val()
    if (!content.trim() && !topic.trim()) {
        $("#ai_quiz_status").html("<span class='text-danger'>ใส่เนื้อหา training หรือ topic อย่างน้อยหนึ่งอย่างก่อนครับ</span>")
        return
    }
    var payload = {
        topic: topic,
        content: content,
        language: $("#ai_quiz_language").val(),
        difficulty: $("#ai_quiz_difficulty").val(),
        model: $("#ai_quiz_model").val(),
        count: parseInt($("#ai_quiz_count").val(), 10) || 5
    }
    $("#ai_quiz_status").html("<i class='fa fa-spinner fa-spin'></i> กำลังสร้างด้วย " + escapeHtml(payload.model) + "… อาจใช้เวลา 10–30 วินาที")
    $("#ai_quiz_generate").prop("disabled", true).html("<i class='fa fa-spinner fa-spin'></i> Generating")
    query("/ai/quiz", "POST", payload, false)
        .success(function (questions) {
            $("#ai_quiz_generate").prop("disabled", false).html("<i class='fa fa-magic'></i> Generate Quiz")
            if (!questions || questions.length === 0) {
                $("#ai_quiz_status").html("<span class='text-warning'>AI ไม่ได้สร้างคำถาม ลองใหม่อีกครั้ง</span>")
                return
            }
            $.each(questions, function (i, q) {
                addQuestion(q)
            })
            $("#ai_quiz_status").html("<span class='text-success'><i class='fa fa-check'></i> เพิ่ม " + questions.length + " ข้อแล้ว</span>")
        })
        .error(function (data) {
            $("#ai_quiz_generate").prop("disabled", false).html("<i class='fa fa-magic'></i> Generate Quiz")
            var msg = (data && data.responseJSON && data.responseJSON.message) ? data.responseJSON.message : "เกิดข้อผิดพลาดในการเรียก AI"
            $("#ai_quiz_status").html("<span class='text-danger'>" + escapeHtml(msg) + "</span>")
        })
}

// save attempts to POST or PUT a training page to /api/training/
function save(idx) {
    var page = {}
    page.name = $("#name").val()
    editor = CKEDITOR.instances["html_editor"]
    page.html = editor.getData()
    var quiz = collectQuiz()
    page.quiz = quiz.length > 0 ? JSON.stringify(quiz) : ""
    if (idx != -1) {
        page.id = pages[idx].id
        api.trainingId.put(page)
            .success(function (data) {
                successFlash("Training page edited successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    } else {
        api.training.post(page)
            .success(function (data) {
                successFlash("Training page added successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    }
}

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#name").val("")
    $("#html_editor").val("")
    $("#quizBuilder").empty()
    $("#ai_quiz_topic").val("")
    $("#ai_quiz_status").empty()
    $("#modal").modal('hide')
}

var deletePage = function (idx) {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the training page. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(pages[idx].name),
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.trainingId.delete(pages[idx].id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                'Training Page Deleted!',
                'This training page has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

function edit(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(idx)
    })
    $("#html_editor").ckeditor()
    setupAutocomplete(CKEDITOR.instances["html_editor"])
    var page = {}
    if (idx != -1) {
        $("#modalLabel").text("Edit Training Page")
        page = pages[idx]
        $("#name").val(page.name)
        $("#html_editor").val(page.html)
        renderQuiz(page.quiz)
    } else {
        $("#modalLabel").text("New Training Page")
        renderQuiz("")
    }
}

function copy(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(-1)
    })
    $("#html_editor").ckeditor()
    var page = pages[idx]
    $("#name").val("Copy of " + page.name)
    $("#html_editor").val(page.html)
    renderQuiz(page.quiz)
}

function load() {
    $("#trainingTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    api.training.get()
        .success(function (ps) {
            pages = ps
            $("#loading").hide()
            if (pages.length > 0) {
                $("#trainingTable").show()
                trainingTable = $("#trainingTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                trainingTable.clear()
                pageRows = []
                $.each(pages, function (i, page) {
                    pageRows.push([
                        escapeHtml(page.name),
                        moment(page.modified_date).format('MMMM Do YYYY, h:mm:ss a'),
                        "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Page' onclick='edit(" + i + ")'>\
                    <i class='fa fa-pencil'></i>\
                    </button></span>\
                    <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Page' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>\
                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Page' onclick='deletePage(" + i + ")'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                })
                trainingTable.rows.add(pageRows).draw()
                $('[data-toggle="tooltip"]').tooltip()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching training pages")
        })
}

$(document).ready(function () {
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    });
    // Quiz builder controls (delegated so they work for dynamic questions)
    $("#addQuestionBtn").on("click", function () {
        addQuestion()
    })
    $("#ai_quiz_generate").on("click", aiGenerateQuiz)
    $("#quizBuilder").on("click", ".gfq-add-choice", function () {
        addChoice($(this).closest(".gf-quiz-q"), "", false)
    })
    $("#quizBuilder").on("click", ".gfq-del-choice", function () {
        $(this).closest(".gfq-choice").remove()
    })
    $("#quizBuilder").on("click", ".gfq-del-q", function () {
        $(this).closest(".gf-quiz-q").remove()
    })
    CKEDITOR.on('dialogDefinition', function (ev) {
        var dialogName = ev.data.name;
        var dialogDefinition = ev.data.definition;
        if (dialogName == 'link') {
            dialogDefinition.minWidth = 500
            dialogDefinition.minHeight = 100
            var infoTab = dialogDefinition.getContents('info');
            infoTab.get('linkType').hidden = true;
        }
    });
    load()
})
