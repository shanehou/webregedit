$(function () {
    var registry = {};

    var importModal = $('#importModal').modal({
        backdrop: false,
        keyboard: false,
        show: false
    });
    var outputModal = $('#outputModal').modal({
        backdrop: false,
        keyboard: false,
        show: false
    });
    var errorModal = $('#errorModal').modal({
        show: false
    });
    var attrAddModal = $('#attrAddModal').modal({
        show: false
    });
    var attrEditModal = $('#attrEditModal').modal({
        show: false
    });

    $('#importForm').on('submit', function(event) {
        event.preventDefault();
        $('#importFile').click();
    });
    $('#import').on('click', function(event) {
        event.preventDefault();
        importModal.modal('show');
    });
    $('#importFile').on('click', function(event) {
        var $this = $(this);
        var $registryFile = $('#inputRegistryFile');
        event.preventDefault();
        $.ajax({
            beforeSend: function(jqXHR, settings) {
                $registryFile.parent('.form-group').removeClass('has-error');
                $registryFile.next('.help-block').text('Importing... Please wait...');
                $this.prop('disabled', true);
                $registryFile.prop('disabled', true);
            },
            url: '/import/',
            type: 'POST',
            dataType: 'json',
            data: {registry: $registryFile.val()},
        }).done(function(data, textStatus, jqXHR) {
            if (data.err) {
                $registryFile.parent('.form-group').addClass('has-error');
                $registryFile.next('.help-block').text(data.err);
                $this.prop('disabled', false);
                $registryFile.prop('disabled', false);
            } else {
                location.reload(true);
            }
        }).fail(function(jqXHR, textStatus, errorThrown) {
            importModal.modal('hide');
            $('#errorModal p').text(textStatus);
            errorModal.modal('show');
        });
    });

    $('#outputForm').on('submit', function(event) {
        event.preventDefault();
        $('#outputFile').click();
    });
    $('#output').on('click', function(event) {
        event.preventDefault();
        outputModal.modal('show');
    });
    $('#outputFile').on('click', function(event) {
        var $this = $(this);
        var $registryFile = $('#outputRegistryFile');
        event.preventDefault();
        $.ajax({
            beforeSend: function(jqXHR, settings) {
                $registryFile.parent('.form-group').removeClass('has-error');
                $registryFile.next('.help-block').text('Outputing... Please wait...');
                $this.prop('disabled', true);
                $registryFile.prop('disabled', true);
            },
            url: '/output/',
            type: 'POST',
            dataType: 'json',
            data: {registry: $registryFile.val()},
        }).done(function(data, textStatus, jqXHR) {
            if (data.err) {
                $registryFile.parent('.form-group').addClass('has-error');
                $registryFile.next('.help-block').text(data.err);
            } else {
                $registryFile.next('.help-block').text("");
                outputModal.modal('hide');
            }
            $this.prop('disabled', false);
            $registryFile.prop('disabled', false);
        }).fail(function(jqXHR, textStatus, errorThrown) {
            outputModal.modal('hide');
            $('#errorModal p').text(textStatus);
            errorModal.modal('show');
        });
    });

    $('#drop').on('click', function(event) {
        event.preventDefault();
        if (confirm("Do you want to drop the whole registry?")) {
            $.ajax({
                url: '/drop/',
                type: 'POST',
                dataType: 'json'
            }).done(function(data, textStatus, jqXHR) {
                if (data.err) {
                    $('#errorModal p').text(data.err);
                    errorModal.modal('show');
                } else {
                    location.reload(true);
                }
            }).fail(function(jqXHR, textStatus, errorThrown) {
                attrAddModal.modal('hide');
                $('#errorModal p').text(textStatus);
                errorModal.modal('show');
            });
        }
    });

    var trimQuotes = function(str) {
        if (str.charAt(0) == '"' && str.charAt(str.length-1) == '"') {
            return str.substring(1, str.length-1);
        }
        return str;
    }

    var treeview = $('#treeview').jstree({
        'core': {
            'data': {
                'url': function(node) {
                    return '/read/' + (node.id === '#' ? 'root' : node.id);
                },
                'dataType': 'json',
                'success': function(nodes) {
                    if (nodes) {
                        for (var i in nodes) {
                            nodes[i].a_attr = {title: nodes[i].text, path: nodes[i].path};
                            var regNode = {};
                            var nodeAttr = nodes[i].attr;
                            for (var j in nodeAttr) {
                                regNode[trimQuotes(j)] = nodeAttr[j];
                            }
                            registry[nodes[i].id] = regNode;
                        }
                    } else {
                        $('#import').click();
                    }
                }
            },
            'check_callback' : function (operation, node, node_parent, node_position, more) {
                switch (operation) {
                    case 'create_node':
                        return true;
                    case 'rename_node':
                        return true;
                    case 'delete_node':
                        return true;
                    default:
                        $('#errorModal p').text('Invalid operation!');
                        errorModal.modal('show');
                        return false;
                }
            }
        },
        'search': {
            fuzzy: false
        },
        'plugins': [
            "contextmenu", "search"
        ]
    }).on('select_node.jstree', function(event, params) {
        var attrs = $('#attributes > tbody');
        attrs.empty();
        var selectedNode = registry[params.node.id];
        for (var i in selectedNode) {
            attrs.append('<tr><td>'+i+'</td><td>'+selectedNode[i].type+'</td><td>'+selectedNode[i].data+'</td><td><span class="glyphicon glyphicon-edit"></span>&nbsp;<span class="glyphicon glyphicon-trash"></span></td></tr>');
        }
        attrs.append('<tr><td>-</td><td>-</td><td>-</td><td><span class="glyphicon glyphicon-plus"></span></td></tr>');
    }).on('create_node.jstree', function(event, params) {
        $.ajax({
            url: '/node/create',
            type: 'POST',
            dataType: 'json',
            data: {"objectId": params.parent, "text": params.node.text},
            success: function(data, textStatus, jqXHR){
                if (data.err) {
                    $('#errorModal p').text(data.err);
                    errorModal.modal('show');
                }
                params.instance.set_id(params.node, data.id);
            },
            error: function(jqXHR, textStatus, errorThrown) {
                if (textStatus) {
                    $('#errorModal p').text(textStatus);
                    errorModal.modal('show');
                }
            }
        });
    }).on('rename_node.jstree', function(event, params) {
        $.ajax({
            url: '/node/rename',
            type: 'POST',
            dataType: 'json',
            data: {"objectId": params.node.id, "text": params.text},
            success: function(data, textStatus, jqXHR){
                if (data.err) {
                    $('#errorModal p').text(data.err);
                    errorModal.modal('show');
                }
            },
            error: function(jqXHR, textStatus, errorThrown) {
                if (textStatus) {
                    $('#errorModal p').text(textStatus);
                    errorModal.modal('show');
                }
            }
        });
    }).on('delete_node.jstree', function(event, params) {
        $.ajax({
            url: '/node/delete',
            type: 'POST',
            dataType: 'json',
            data: {objectId: params.node.id, text: params.node.text},
            success: function(data, textStatus, jqXHR){
                if (data.err) {
                    $('#errorModal p').text(data.err);
                    errorModal.modal('show');
                }
            },
            error: function(jqXHR, textStatus, errorThrown) {
                if (textStatus) {
                    $('#errorModal p').text(textStatus);
                    errorModal.modal('show');
                }
            }
        });
    }).on('open_all.jstree', function(event, params) {
        params.instance.close_all();
    }).on('close_all.jstree', function(event, params) {
        params.instance.search($('#searchContent').val());
    });

    var $valueEdit = $('#valueEdit');
    var $hiddenValueEdit = $('#hiddenValueEdit');
    var $dataEditInput = $('#dataEditInput');
    var $dataEditTextarea = $('#dataEditTextarea');
    var $attrEdit = $('#attrEdit');
    $attrEdit.on('submit', function(event) {
        event.preventDefault();
        $('#submitEdit').click();
    });
    var $attr, $value, $valueType, $valueData;
    $('#attributes').on('click', 'span.glyphicon.glyphicon-edit', function(event) {
        event.preventDefault();
        $attr = $(this).parent('td').siblings('td');
        $value = $attr.first();
        $valueType = $value.next('td');
        $valueData = $valueType.next('td');
        $valueEdit.val($value.text());
        $hiddenValueEdit.val($value.text());
        switch ($valueType.text()) {
            case "REG_SZ":
                $dataEditInput.val($valueData.text()).show();
                $dataEditTextarea.hide().val("");
                $dataEditInput.off('keypress');
                break;
            case "REG_DWORD":
                $dataEditInput.on('keypress', function(event) {
                    if ((event.which < 48) || (event.which > 57 && event.which < 65) || (event.which > 70 && event.which < 97) || (event.which > 102)) {
                        return false;
                    }
                    return true;
                });
                $dataEditInput.val($valueData.text()).show();
                $dataEditTextarea.hide().val("");
                break;
            default:
                $dataEditTextarea.on('keypress', function(event) {
                    if ((event.which < 44) || (event.which > 44 && event.which < 48) || (event.which > 57 && event.which < 65) || (event.which > 70 && event.which < 97) || (event.which > 102)) {
                        return false;
                    }
                    return true;
                });
                $dataEditTextarea.val($valueData.text()).show();
                $dataEditInput.hide().val("");
        }
        attrEditModal.modal('show');
    }).on('hidden.bs.modal', function(event) {
        $valueEdit.val("");
        $hiddenValueEdit.val("");
        $dataEditInput.val("");
        $dataEditTextarea.text("");
        $attrEdit.children('.form-group').first().removeClass('has-error');
        $valueEdit.next('.help-block').empty();
    });

    $('#submitEdit').on('click', function(event) {
        event.preventDefault();
        var id = treeview.jstree(true).get_selected()[0];
        $.ajax({
            url: '/attr/edit',
            type: 'POST',
            dataType: 'json',
            data: {objectId: id, orgiValue: $hiddenValueEdit.val(), value: $valueEdit.val(), valueData: $dataEditInput.val()||$dataEditTextarea.val()},
        }).done(function(data, textStatus, jqXHR) {
            if (data.err) {
                switch (data.err) {
                    case "value already exists!":
                        $attrEdit.children('.form-group').first().addClass('has-error');
                        $valueEdit.next('.help-block').text(data.err);
                        break;
                    default:
                        attrEditModal.modal('hide');
                        $('#errorModal p').text(data.err);
                        errorModal.modal('show');
                }
            } else {
                var origType = registry[id][$hiddenValueEdit.val()].type;
                delete registry[id][$hiddenValueEdit.val()];
                registry[id][$valueEdit.val()] = {data: $dataEditInput.val()||$dataEditTextarea.val(), type: origType};
                $value.text($valueEdit.val());
                $valueData.text($dataEditInput.val()||$dataEditTextarea.val());
                attrEditModal.modal('hide');
            }
        }).fail(function(jqXHR, textStatus, errorThrown) {
            attrEditModal.modal('hide');
            $('#errorModal p').text(textStatus);
            errorModal.modal('show');
        });
    });


    var $valueAdd = $('#valueAdd');
    var $dataAddInput = $('#dataAddInput');
    var $dataAddTextarea = $('#dataAddTextarea');
    var $attrAdd = $('#attrAdd');
    $attrAdd.on('submit', function(event) {
        event.preventDefault();
        $('#submitAdd').click();
    });
    var $valueType;
    $(':radio[name="valueType"]', $attrAdd).on('change', function(event) {
        $valueType = this.value;
        switch ($valueType) {
            case "REG_SZ":
                $dataAddInput.show().val("");
                $dataAddTextarea.hide().val("");
                $dataAddInput.off('keypress');
                break;
            case "REG_DWORD":
                $dataAddInput.on('keypress', function(event) {
                    if ((event.which < 48) || (event.which > 57 && event.which < 65) || (event.which > 70 && event.which < 97) || (event.which > 102)) {
                        return false;
                    }
                    return true;
                });
                $dataAddInput.show().val("");
                $dataAddTextarea.hide().val("");
                break;
            default:
                $dataAddTextarea.on('keypress', function(event) {
                    if ((event.which < 44) || (event.which > 44 && event.which < 48) || (event.which > 57 && event.which < 65) || (event.which > 70 && event.which < 97) || (event.which > 102)) {
                        return false;
                    }
                    return true;
                });
                $dataAddTextarea.show().val("");
                $dataAddInput.hide().val("");
        }
    });

    $('#attributes').on('click', 'span.glyphicon.glyphicon-plus', function(event) {
        event.preventDefault();
        $(':radio[name="valueType"]:first', $attrAdd).change();
        attrAddModal.modal('show');
    }).on('hidden.bs.modal', function(event) {
        $valueAdd.val("");
        $dataAddInput.val("");
        $dataAddTextarea.text("");
        $attrAdd.children('.form-group').first().removeClass('has-error');
        $valueAdd.next('.help-block').empty();
    });


    $('#submitAdd').on('click', function(event) {
        event.preventDefault();
        var id = treeview.jstree(true).get_selected()[0];
        $.ajax({
            url: '/attr/add',
            type: 'POST',
            dataType: 'json',
            data: {objectId: id, valueType: $valueType, value: $valueAdd.val(), valueData: $dataAddInput.val()||$dataAddTextarea.val()},
        }).done(function(data, textStatus, jqXHR) {
            if (data.err) {
                switch (data.err) {
                    case "value already exists!":
                        $attrAdd.children('.form-group').first().addClass('has-error');
                        $valueAdd.next('.help-block').text(data.err);
                        break;
                    default:
                        attrAddModal.modal('hide');
                        $('#errorModal p').text(data.err);
                        errorModal.modal('show');
                }
            } else {
                registry[id][$valueAdd.val()] = {data: $dataAddInput.val()||$dataAddTextarea.val(), type: $valueType};
                $('<tr><td>'+$valueAdd.val()+'</td><td>'+$valueType+'</td><td>'+($dataAddInput.val()||$dataAddTextarea.val())+'</td><td><span class="glyphicon glyphicon-edit"></span>&nbsp;<span class="glyphicon glyphicon-trash"></span></td></tr>').insertBefore($('#attributes .glyphicon.glyphicon-plus').parent('td').parent('tr'));
                attrAddModal.modal('hide');
            }
        }).fail(function(jqXHR, textStatus, errorThrown) {
            attrAddModal.modal('hide');
            $('#errorModal p').text(textStatus);
            errorModal.modal('show');
        });
    });

    $('#attributes').on('click', 'span.glyphicon.glyphicon-trash', function(event) {
        event.preventDefault();
        var id = treeview.jstree(true).get_selected()[0];
        var $this = $(this);
        $attr = $this.parent('td').siblings('td');
        $value = $attr.first();
        if (confirm("Delete this value?")) {
            $.ajax({
                url: '/attr/delete',
                type: 'POST',
                dataType: 'json',
                data: {objectId: id, value: $value.text()},
            }).done(function(data, textStatus, jqXHR) {
                if (data.err) {
                    $('#errorModal p').text(data.err);
                    errorModal.modal('show');
                } else {
                    delete registry[id][$value.text()];
                    $this.parent('td').parent('tr').remove();
                }
            }).fail(function(jqXHR, textStatus, errorThrown) {
                $('#errorModal p').text(textStatus);
                errorModal.modal('show');
            });
        }
    });

    $('#search').on('submit', function(event) {
        event.preventDefault();
        treeview.jstree(true).open_all(null, 0);
    });
});