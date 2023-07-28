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