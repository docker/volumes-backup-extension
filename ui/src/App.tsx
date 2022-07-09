import React, {useContext, useEffect} from "react";
import {DataGrid, GridActionsCellItem, GridCellParams,} from "@mui/x-data-grid";
import {createDockerDesktopClient} from "@docker/extension-api-client";
import {Backdrop, Box, CircularProgress, Grid, LinearProgress, Stack, Tooltip, Typography,} from "@mui/material";
import {
    ArrowCircleDown as ArrowCircleDownIcon,
    CopyAll as CopyAllIcon,
    Delete as DeleteIcon,
    DeleteForever as DeleteForeverIcon,
    Devices as DevicesIcon,
    Download as DownloadIcon,
    ExitToApp as ExitToAppIcon,
    Layers as LayersIcon,
    PlayArrow as PlayArrowIcon,
    Upload as UploadIcon,
} from "@mui/icons-material";
import ExportDialog from "./components/ExportDialog";
import ImportDialog from "./components/ImportDialog";
import SaveDialog from "./components/SaveDialog";
import LoadDialog from "./components/LoadDialog";
import CloneDialog from "./components/CloneDialog";
import TransferDialog from "./components/TransferDialog";
import RunContainerDialog from "./components/RunContainerDialog";
import DeleteForeverDialog from "./components/DeleteForeverDialog";
import {MyContext} from ".";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
    return client;
}

export function App() {
    const context = useContext(MyContext);
    const [rows, setRows] = React.useState([]);
    const [volumeContainersMap, setVolumeContainersMap] = React.useState<Record<string, string[]>>({});
    const [volumeSizeMap, setVolumeSizeMap] = React.useState<Record<string, string>>({});
    const [volumes, setVolumes] = React.useState([]);
    const [reloadTable, setReloadTable] = React.useState<boolean>(false);
    const [loadingVolumes, setLoadingVolumes] = React.useState<boolean>(true);
    const [volumesSizeLoadingMap, setVolumesSizeLoadingMap] = React.useState<Record<string, boolean>>({});

    const [actionInProgress, setActionInProgress] =
        React.useState<boolean>(false);

    const [openExportDialog, setOpenExportDialog] =
        React.useState<boolean>(false);
    const [openImportDialog, setOpenImportDialog] =
        React.useState<boolean>(false);
    const [openSaveDialog, setOpenSaveDialog] = React.useState<boolean>(false);
    const [openLoadDialog, setOpenLoadDialog] = React.useState<boolean>(false);
    const [openCloneDialog, setOpenCloneDialog] = React.useState<boolean>(false);
    const [openTransferDialog, setOpenTransferDialog] =
        React.useState<boolean>(false);
    const [openRunContainerDialog, setOpenRunContainerDialog] =
        React.useState<boolean>(false);
    const [openDeleteForeverDialog, setOpenDeleteForeverDialog] =
        React.useState<boolean>(false);
    const ddClient = useDockerDesktopClient();

    const columns = [
        {field: "id", headerName: "ID", width: 70, hide: true},
        {field: "volumeDriver", headerName: "Driver"},
        {
            field: "volumeName",
            headerName: "Volume name",
            flex: 1,
        },
        {field: "volumeLinks", hide: true},
        {
            field: "volumeContainers",
            headerName: "Containers",
            flex: 1,
            renderCell: (params) => {
                if (params.row.volumeContainers) {
                    return (
                        <Box display="flex" flexDirection="column">
                            {params.row.volumeContainers.map((container) => (
                                <Typography key={container}>{container}</Typography>
                            ))}
                        </Box>
                    );
                }
                return <></>;
            },
        },
        {
            field: "volumeSize",
            headerName: "Size",
            renderCell: (params) => {
                if (volumesSizeLoadingMap[params.row.volumeName]) {
                    return (<Box sx={{width: '100%'}}>
                        <LinearProgress/>
                    </Box>)
                }
                return <Typography>{params.row.volumeSize}</Typography>
            },
        },
        {
            field: "actions",
            type: "actions",
            headerName: "Actions",
            minWidth: 220,
            sortable: false,
            flex: 1,
            getActions: (params) => [
                <GridActionsCellItem
                    key={"action_view_volume_" + params.row.id}
                    icon={
                        <Tooltip title="View volume">
                            <ExitToAppIcon>View volume</ExitToAppIcon>
                        </Tooltip>
                    }
                    label="View volume"
                    onClick={handleNavigate(params.row)}
                    disabled={actionInProgress}
                    showInMenu
                />,
                <GridActionsCellItem
                    key={"action_run_container_from_volume_" + params.row.id}
                    icon={
                        <Tooltip title="Run container from volume">
                            <PlayArrowIcon>Run container from volume</PlayArrowIcon>
                        </Tooltip>
                    }
                    label="Run container from volume"
                    onClick={handleRunContainer(params.row)}
                    disabled={actionInProgress}
                />,
                <GridActionsCellItem
                    key={"action_clone_volume_" + params.row.id}
                    icon={
                        <Tooltip title="Clone volume">
                            <CopyAllIcon>Clone volume</CopyAllIcon>
                        </Tooltip>
                    }
                    label="Clone volume"
                    onClick={handleClone(params.row)}
                    disabled={actionInProgress}
                />,
                <GridActionsCellItem
                    key={"action_export_" + params.row.id}
                    icon={
                        <Tooltip title="Export volume">
                            <DownloadIcon>Export volume</DownloadIcon>
                        </Tooltip>
                    }
                    label="Export volume"
                    onClick={handleExport(params.row)}
                    disabled={params.row.volumeSize === "0B"}
                />,
                <GridActionsCellItem
                    key={"action_import_" + params.row.id}
                    icon={
                        <Tooltip title="Import gzip'ed tarball">
                            <UploadIcon>Import gzip'ed tarball</UploadIcon>
                        </Tooltip>
                    }
                    label="Import gzip'ed tarball"
                    onClick={handleImport(params.row)}
                />,
                <GridActionsCellItem
                    key={"action_save_" + params.row.id}
                    icon={
                        <Tooltip title="Save to image">
                            <LayersIcon>Save to image</LayersIcon>
                        </Tooltip>
                    }
                    label="Save to image"
                    onClick={handleSave(params.row)}
                    showInMenu
                    disabled={params.row.volumeSize === "0B"}
                />,
                <GridActionsCellItem
                    key={"action_load_" + params.row.id}
                    icon={
                        <Tooltip title="Load from image">
                            <ArrowCircleDownIcon>Load from image</ArrowCircleDownIcon>
                        </Tooltip>
                    }
                    label="Load from image"
                    onClick={handleLoad(params.row)}
                    showInMenu
                />,
                <GridActionsCellItem
                    key={"action_transfer_" + params.row.id}
                    icon={
                        <Tooltip title="Transfer to host">
                            <DevicesIcon>Transfer to host</DevicesIcon>
                        </Tooltip>
                    }
                    label="Transfer to host"
                    onClick={handleTransfer(params.row)}
                    showInMenu
                    disabled={params.row.volumeSize === "0B"}
                />,
                <GridActionsCellItem
                    key={"action_empty_" + params.row.id}
                    icon={
                        <Tooltip title="Empty volume">
                            <DeleteIcon>Empty volume</DeleteIcon>
                        </Tooltip>
                    }
                    label="Empty volume"
                    onClick={handleEmpty(params.row)}
                    showInMenu
                    disabled={params.row.volumeSize === "0B"}
                />,
                <GridActionsCellItem
                    key={"action_delete_" + params.row.id}
                    icon={
                        <Tooltip title="Delete volume">
                            <DeleteForeverIcon>Delete volume</DeleteForeverIcon>
                        </Tooltip>
                    }
                    label="Delete volume"
                    onClick={handleDelete(params.row)}
                    showInMenu
                />,
            ],
        },
    ];

    const handleNavigate = (row) => async () => {
        ddClient.desktopUI.navigate.viewVolume(row.volumeName);
    };

    const handleRunContainer = (row) => () => {
        setOpenRunContainerDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleClone = (row) => () => {
        setOpenCloneDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleExport = (row) => () => {
        setOpenExportDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleImport = (row) => () => {
        setOpenImportDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleSave = (row) => () => {
        setOpenSaveDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleLoad = (row) => async () => {
        setOpenLoadDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleTransfer = (row) => async () => {
        setOpenTransferDialog(true);
        context.actions.setVolumeName(row.volumeName);
    };

    const handleEmpty = (row) => async () => {
        await emptyVolume(row.volumeName);
        await calculateVolumeSize(row.volumeName);
    };

    const handleDelete = (row) => async () => {
        setOpenDeleteForeverDialog(true)
        context.actions.setVolumeName(row.volumeName);
    };

    const handleCellClick = (params: GridCellParams) => {
        if (params.colDef.field === "volumeName") {
            ddClient.desktopUI.navigate.viewVolume(params.row.volumeName);
        }
    };

    const calculateVolumeSize = async (volumeName: string) => {
        let volumesSizeLoadingMapCopy = volumesSizeLoadingMap
        volumesSizeLoadingMapCopy[volumeName] = true
        setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy)

        try {
            const size = await computeVolumeSize(volumeName)

            let rowsCopy = rows.slice() // copy the array
            const index = rowsCopy.findIndex(element => element.volumeName === volumeName)
            rowsCopy[index].volumeSize = size

            setRows(rowsCopy)
        } catch (error) {
            ddClient.desktopUI.toast.error(
                `Failed to recalculate volume size: ${error.stderr}`
            );
        } finally {
            let volumesSizeLoadingMapCopy = volumesSizeLoadingMap
            volumesSizeLoadingMapCopy[volumeName] = false
            setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy)
        }
    };

    const computeVolumeSize = async (volumeName: string): Promise<string> => {
        let size = "-"
        const tmpDir = "/recalc-vol-size"

        // e.g. docker run --rm -v postgres-vol:/pgdata alpine /bin/sh -c  "du -d 0 -h /pgdata | cut -f 1"
        // get only dir size, without the name, e.g:
        // 41.5M
        // instead of
        // 41.5M	/pgdata
        const args = [
            "--rm",
            `-v=${volumeName}:${tmpDir}`,
            "alpine",
            "/bin/sh",
            "-c",
            `"du -d 0 -h ${tmpDir}"`
        ];
        const result = await ddClient.docker.cli.exec("run", args);

        if (result.stderr !== "") {
            ddClient.desktopUI.toast.error(result.stderr);
        } else {
            const s = result.lines()[0].split("\t"); // e.g. 41.5M	/recalc-vol-size
            size = s[0]

            if (size === "4.0K") {
                // If a directory size is 4K, it is in fact "empty".
                // The metadata of the folder is stored in blocks and 4K is the minimum filesystem's block size.
                // Therefore, we set it to "0B" to indicate that the directory is empty.
                size = "0B"
            }
        }
        return size
    }

    // This useEffect will pull the alpine image as it is needed to compute each volume size.
    useEffect(() => {
        const pullAlpineImage = async() => {
            const startTime = performance.now()

            const result = await ddClient.docker.cli.exec("pull", [
                "alpine",
            ]);

            if (result.stderr !== "") {
                ddClient.desktopUI.toast.error(result.stderr);
                return
            }

            const endTime = performance.now()
            console.log(`[pullAlpineImage] took ${endTime - startTime} ms.`)
        }

        pullAlpineImage()
    }, [])

    useEffect(() => {
        const listVolumes = async () => {
            const startTime = performance.now()
            setLoadingVolumes(true);
            try {
                const result = await ddClient.docker.cli.exec("volume", [
                    "ls",
                    "--format",
                    '"{{ json . }}"',
                ]);

                if (result.stderr !== "") {
                    ddClient.desktopUI.toast.error(result.stderr);
                } else {
                    const volumes = result.parseJsonLines();
                    const promises = volumes.map((volume) =>
                        getContainersForVolume(volume.Name)
                    );

                    Promise.allSettled(promises)
                        .then((values) => {
                            const vcMap = {};
                            const vSMap = {};

                            values.forEach((value) => {
                                if (value.status === "rejected") {
                                    ddClient.desktopUI.toast.error(value.reason);
                                    return;
                                }

                                const {volumeName, containers, volumeSize} = value.value;
                                vcMap[volumeName] = containers;
                                vSMap[volumeName] = volumeSize;
                            });

                            setVolumeContainersMap(vcMap);
                            setVolumeSizeMap(vSMap);
                        })

                        .finally(() => {
                            setLoadingVolumes(false);
                            const endTime = performance.now()
                            console.log(`[listVolumes] took ${endTime - startTime} ms.`)
                        });

                    setVolumes(volumes);
                }
            } catch (error) {
                setLoadingVolumes(false);
                ddClient.desktopUI.toast.error(
                    `Failed to list volumes: ${error.stderr}`
                );
            }
        };

        listVolumes();

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [reloadTable]);

    useEffect(() => {
        const startTime = performance.now()

        const rows = volumes
            .sort((a, b) => a.Name.localeCompare(b.Name))
            .map((volume, index) => {
                return {
                    id: index,
                    volumeDriver: volume.Driver,
                    volumeName: volume.Name,
                    volumeLinks: volume.Links,
                    volumeContainers: volumeContainersMap[volume.Name],
                    volumeSize: volumeSizeMap[volume.Name],
                };
            });

        setRows(rows);

        const endTime = performance.now()
        console.log(`[setRows] took ${endTime - startTime} ms.`)
    }, [volumeContainersMap, volumeSizeMap]);

    const emptyVolume = async (volumeName: string) => {
        setActionInProgress(true);

        try {
            const output = await ddClient.docker.cli.exec("run", [
                "--rm",
                `-v=${volumeName}:/vackup-volume `,
                "busybox",
                "/bin/sh",
                "-c",
                '"rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/*"', // hidden and not-hidden files and folders: .[!.]* matches all dot files except . and files whose name begins with .., and ..?* matches all dot-dot files except ..
            ]);
            if (output.stderr !== "") {
                ddClient.desktopUI.toast.error(output.stderr);
                return;
            }
            ddClient.desktopUI.toast.success(
                `The content of volume ${volumeName} has been removed`
            );
        } catch (error) {
            ddClient.desktopUI.toast.error(
                `Failed to empty volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
            );
        } finally {
            setActionInProgress(false);
        }
    };

    const getContainersForVolume = async (
        volumeName: string
    ): Promise<{ volumeName: string; containers: string[]; volumeSize: string; }> => {
        try {
            const output = await ddClient.docker.cli.exec("ps", [
                "-a",
                `--filter="volume=${volumeName}"`,
                `--format="{{ .Names}}"`,
            ]);

            if (output.stderr !== "") {
                ddClient.desktopUI.toast.error(output.stderr);
            }

            const volumeSize = await computeVolumeSize(volumeName)

            return {volumeName, containers: output.stdout.trim().split(" "), volumeSize};
        } catch (error) {
            const errorMsg = `Failed to get containers for volume ${volumeName}: ${error.stderr} Error code: ${error.code}`;
            return Promise.reject(errorMsg);
        }
    };

    const handleRunContainerDialogClose = () => {
        setOpenRunContainerDialog(false);
    };

    const handleExportDialogClose = () => {
        setOpenExportDialog(false);
    };

    const handleImportDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenImportDialog(false);
        if (actionSuccessfullyCompleted) {
            calculateVolumeSize(context.store.volumeName);
        }
    };

    const handleSaveDialogClose = () => {
        setOpenSaveDialog(false);
    };

    const handleLoadDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenLoadDialog(false);
        if (actionSuccessfullyCompleted) {
            calculateVolumeSize(context.store.volumeName);
        }
    };

    const handleCloneDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenCloneDialog(false);
        if (actionSuccessfullyCompleted) {
            setReloadTable(!reloadTable);
        }
    };

    const handleTransferDialogClose = () => {
        setOpenTransferDialog(false);
    };

    const handleDeleteForeverDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenDeleteForeverDialog(false);
        if (actionSuccessfullyCompleted) {
            setReloadTable(!reloadTable);
        }
    };

    return (
        <>
            <Typography variant="h3">Vackup Extension</Typography>
            <Typography variant="body1" color="text.secondary" sx={{mt: 2}}>
                Easily backup and restore docker volumes.
            </Typography>
            <Stack direction="column" alignItems="start" spacing={2} sx={{mt: 4}}>
                <Grid container>
                    <Grid item flex={1}>
                        <Backdrop
                            sx={{
                                backgroundColor: "rgba(245,244,244,0.4)",
                                zIndex: (theme) => theme.zIndex.drawer + 1,
                            }}
                            open={actionInProgress}
                        >
                            <CircularProgress color="info"/>
                        </Backdrop>
                        <DataGrid
                            loading={loadingVolumes}
                            components={{
                                LoadingOverlay: LinearProgress,
                            }}
                            rows={rows}
                            columns={columns}
                            pageSize={10}
                            rowsPerPageOptions={[10]}
                            checkboxSelection={false}
                            disableSelectionOnClick={true}
                            autoHeight
                            getRowHeight={() => "auto"}
                            onCellClick={handleCellClick}
                            sx={{
                                "&.MuiDataGrid-root--densityCompact .MuiDataGrid-cell": {
                                    py: 1,
                                },
                                "&.MuiDataGrid-root--densityStandard .MuiDataGrid-cell": {
                                    py: 1,
                                },
                                "&.MuiDataGrid-root--densityComfortable .MuiDataGrid-cell": {
                                    py: 2,
                                },
                            }}
                        />
                    </Grid>

                    {openRunContainerDialog && (
                        <RunContainerDialog
                            open={openRunContainerDialog}
                            onClose={handleRunContainerDialogClose}
                        />
                    )}

                    {openExportDialog && (
                        <ExportDialog
                            open={openExportDialog}
                            onClose={handleExportDialogClose}
                        />
                    )}

                    {openImportDialog && (
                        <ImportDialog
                            open={openImportDialog}
                            onClose={handleImportDialogClose}
                        />
                    )}

                    {openSaveDialog && (
                        <SaveDialog open={openSaveDialog} onClose={handleSaveDialogClose}/>
                    )}

                    {openLoadDialog && (
                        <LoadDialog open={openLoadDialog} onClose={handleLoadDialogClose}/>
                    )}

                    {openCloneDialog && (
                        <CloneDialog
                            open={openCloneDialog}
                            onClose={handleCloneDialogClose}
                        />
                    )}

                    {openTransferDialog && (
                        <TransferDialog
                            open={openTransferDialog}
                            onClose={handleTransferDialogClose}
                        />
                    )}

                    {openDeleteForeverDialog && (
                        <DeleteForeverDialog
                            open={openDeleteForeverDialog}
                            onClose={(e) => {
                                handleDeleteForeverDialogClose(e)
                            }}
                        />
                    )}
                </Grid>
            </Stack>
        </>
    );
}
