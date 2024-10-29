// take the form data and use the patch method to update the settings
function updateSettings(redirect) {
    $("#ajax-error").hide();
    var settings = [];

    // iterate over all input fields in the form and create a json object with the key, value, and type
    $("#settings-form :input").each(function() {
        var key = this.name;
        var type = this.type;
        var value = this.value

        if (type === "file") {
            // just get the filename without the path
            if (this.url) {
                value = this.url
            } else {
                if (redirect !== undefined) {
                    // this is the startup wizard, make sure we have some default value or they won't be created
                    // at all
                    if (this.name === "favicon") {
                        value = "/img/favicon.ico"
                    } else if (this.name === "landing_page_image") {
                        value = "/img/profile.png"
                    }
                } else {
                    // otherwise if the value isn't filled in, leave it what it was so it doesn't get erased
                    return
                }
            }
        } else if (type === "submit") {
            return
        }
        settings.push(
            {"key": key, "value": value, "type": type}
        )
    });
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

            if (redirect !== undefined) {
                window.location = redirect;
            }
        },
        error: function(jqXHR, textStatus, errorThrown) {
            // show #ajax-error with the error message
            $("#ajax-error").html("ERROR: " + textStatus + " " + errorThrown).show();
            $("#ajax-error").removeClass("alert-success").addClass("alert-danger");
        },
        data: JSON.stringify(settings)
    });
}

// this is mostly used by the settings page to upload a file. Post uploads use a different method since they can
// be pasted directly into the editor, however, on the backend they use the same post method to save the file
function uploadFile(fileInput) {
    console.log("uploading file: " + fileInput.files[0].name);
    var formData = new FormData();
    var file = fileInput.files[0];
    formData.append("file", file);
    $.ajax({
        url: "/api/v1/upload",
        type: "post",
        data: formData,
        processData: false,
        contentType: false,
        success: function(json) {
            var url = json.filename;
            console.log("uploaded file: " + url);
            // insert the url into the settings form so that when it gets submitted it will be saved in the db
            fileInput.url = url;
        },
        error: function(jqXHR, textStatus, errorThrown) {
            console.log("ERROR: " + textStatus + " " + errorThrown);
        }
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