// take the form data and use the patch method to update the settings
function updateSettings() {
    $("#ajax-error").hide();
    var settings = [];
    for (var i = 0; i < $("#settings-form").serializeArray().length; i++) {
        var key = $("#settings-form").serializeArray()[i].name;
        var value = $("#settings-form").serializeArray()[i].value;
        settings.push(
            {"key": key, "value": value}
        )
    }
    console.log("PATCH: " + JSON.stringify(settings));

    $.ajax({
        url: "/api/v1/settings",
        type: "patch",
        dataType: "json",
        contentType: "application/json",
        success: function(json) {
            // change #ajax-error to success and show "settings updated"
            $("#ajax-error").html("Settings updated").show();
            $("#ajax-error").removeClass("alert-danger").addClass("alert-success");
        },
        error: function(jqXHR, textStatus, errorThrown) {
            // show #ajax-error with the error message
            $("#ajax-error").html("ERROR: " + textStatus + " " + errorThrown).show();
            $("#ajax-error").removeClass("alert-success").addClass("alert-danger");
        },
        data: JSON.stringify(settings)
    });
}

function updatePost(id, publish) {
    var tags = $("#tags").text().split(',')
    for (var i = 0; i < tags.length; i++) {
        tags[i] = {"name": tags[i].trim()}
    }
    var time = $("#created_at").val();
    var vtime = moment.utc(time)

    var post = {
        "id": id,
        "created_at": vtime,
        "title": $("#title").text(),
        "content": simplemde.value(),
        "tags": tags,
        "slug": $("#slug").text(),
        "draft": !publish,
    }
    console.log("PATCH: " + JSON.stringify(post));

    $.ajax({
        url: "/api/v1/posts",
        type: "patch",
        dataType: "json",
        contentType: "application/json",
        success: function(json) {
            var time = moment(json.created_at);
            var slug = json.slug;
            window.location.href="/admin/posts/" + time.format("YYYY/MM/DD") + "/" + slug;
        },
        data: JSON.stringify(post)
    });
}

function createPost(publish) {
    var tags = $("#tags").val().split(',')
    for (var i = 0; i < tags.length; i++) {
        tags[i] = {"name": tags[i].trim()}
    }

    var time = $("#created_at").val();
    var vtime = moment.utc(time)

    var post = {
        "title": $("#title").val(),
        "created_at": vtime,
        "content": simplemde.value(),
        "tags": tags,
        "draft": !publish,
    }
    console.log("POST: " + JSON.stringify(post));

    $.ajax({
        url: "/api/v1/posts",
        type: "post",
        dataType: "json",
        contentType: "application/json",
        success: function(json) {
            var time = moment(json.created_at);
            var slug = json.slug;
            window.location.href="/posts/" + time.format("YYYY/MM/DD") + "/" + slug;
        },
        error: function(jqXHR, textStatus, errorThrown) {
            alert("ERROR: " + textStatus + " " + errorThrown);
        },
        data: JSON.stringify(post)
    });
}

function publishPost(id) {
    $.ajax({
        url: "/api/v1/publish/" + id,
        type: "patch",
        success: function(json) {
            var time = moment(json.created_at);
            var slug = json.slug;
            window.location.href="/posts/" + time.format("YYYY/MM/DD") + "/" + slug;
        }
    })
}

function draftPost(id) {
    $.ajax({
        url: "/api/v1/draft/" + id,
        type: "patch",
        success: function(json) {
            var time = moment(json.created_at);
            var slug = json.slug;
            window.location.href="/posts/" + time.format("YYYY/MM/DD") + "/" + slug;
        }
    })
}

function deletePost(id) {
    var post = {
        "id": id,
        "title": $("#title").text(),
        "content": $("#content").text(),
    }
    console.log("PATCH: " + JSON.stringify(post));

    $.ajax({
        url: "/api/v1/posts",
        type: "delete",
        dataType: "json",
        contentType: "application/json",
        success: function(json) {
            window.location.href = "/admin/";
        },
        data: JSON.stringify(post)
    });
    return false;
}