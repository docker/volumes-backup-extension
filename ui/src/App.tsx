import React, {useContext, useEffect} from "react";
import {DataGrid, GridActionsCellItem, GridCellParams, GridToolbarColumnsButton, GridToolbarContainer, GridToolbarDensitySelector, GridToolbarFilterButton,} from "@mui/x-data-grid";
import {createDockerDesktopClient} from "@docker/extension-api-client";
import {Backdrop, Box, Button, CircularProgress, Grid, LinearProgress, Stack, Tooltip, Typography,} from "@mui/material";
import {
    ArrowCircleDown as ArrowCircleDownIcon,
    CopyAll as CopyAllIcon,
    Delete as DeleteIcon,
    DeleteForever as DeleteForeverIcon,
    Devices as DevicesIcon,
    Download as DownloadIcon,
    Visibility as VisibilityIcon,
    Layers as LayersIcon,
    PlayArrow as PlayArrowIcon,
    Upload as UploadIcon,
} from "@mui/icons-material";
import ExportDialog from "./components/ExportDialog";
import SaveDialog from "./components/SaveDialog";
import LoadDialog from "./components/LoadDialog";
import CloneDialog from "./components/CloneDialog";
import TransferDialog from "./components/TransferDialog";
import RunContainerDialog from "./components/RunContainerDialog";
import DeleteForeverDialog from "./components/DeleteForeverDialog";
import {MyContext} from ".";
import {isError} from "./common/isError";
import ImportDialog from "./components/ImportDialog";
import { useGetVolumes } from "./hooks/useGetVolumes";
import { Header } from "./components/Header";

const ddClient = createDockerDesktopClient();

function CustomToolbar({openDialog}) {
    return (
      <GridToolbarContainer>
        <Grid container justifyContent="space-between">
            <Grid item>
                <GridToolbarColumnsButton />
                <GridToolbarFilterButton />
                <GridToolbarDensitySelector />
            </Grid>
            <Grid item>
                <Button variant="contained" onClick={openDialog} endIcon={<DownloadIcon />}>Import into new volume</Button>
            </Grid>
        </Grid>
      </GridToolbarContainer>
    );
}
  

export function App() {
    const context = useContext(MyContext);
    const [volumesSizeLoadingMap, setVolumesSizeLoadingMap] = React.useState<Record<string, boolean>>({});

    const [actionInProgress, setActionInProgress] =
        React.useState<boolean>(false);

    const [openExportDialog, setOpenExportDialog] =
        React.useState<boolean>(false);
    const [openImportIntoNewDialog, setOpenImportIntoNewDialog] = React.useState<boolean>(false);
    const [openSaveDialog, setOpenSaveDialog] = React.useState<boolean>(false);
    const [openLoadDialog, setOpenLoadDialog] = React.useState<boolean>(false);
    const [openCloneDialog, setOpenCloneDialog] = React.useState<boolean>(false);
    const [openTransferDialog, setOpenTransferDialog] =
        React.useState<boolean>(false);
    const [openRunContainerDialog, setOpenRunContainerDialog] =
        React.useState<boolean>(false);
    const [openDeleteForeverDialog, setOpenDeleteForeverDialog] =
        React.useState<boolean>(false);

    const columns = [
        {field: "id", headerName: "ID", width: 70, hide: true},
        {field: "volumeDriver", headerName: "Driver", hide: true},
        {
            field: "volumeName",
            headerName: "Volume name",
            flex: 1,
        },
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
                    return (
                        <Box sx={{width: "100%"}}>
                            <LinearProgress/>
                        </Box>
                    );
                }
                return <Typography>{params.row.volumeSize}</Typography>;
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
                            <VisibilityIcon>View volume</VisibilityIcon>
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
                    disabled={params.row.volumeSize === "0 B"}
                />,
                <GridActionsCellItem
                    key={"action_import_" + params.row.id}
                    icon={
                        <Tooltip title="Import">
                            <UploadIcon>Import</UploadIcon>
                        </Tooltip>
                    }
                    label="Import"
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
                    disabled={params.row.volumeSize === "0 B"}
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
                    disabled={params.row.volumeSize === "0 B"}
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
                    disabled={params.row.volumeSize === "0 B"}
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
        context.actions.setVolume(row);
    };

    const handleClone = (row) => () => {
        setOpenCloneDialog(true);
        context.actions.setVolume(row);
    };

    const handleExport = (row) => () => {
        setOpenExportDialog(true);
        context.actions.setVolume(row);
    };

    const handleImport = (row) => () => {
        context.actions.setVolume(row);
        setOpenImportIntoNewDialog(true);
    };

    const handleSave = (row) => () => {
        setOpenSaveDialog(true);
        context.actions.setVolume(row);
    };

    const handleLoad = (row) => async () => {
        setOpenLoadDialog(true);
        context.actions.setVolume(row);
    };

    const handleTransfer = (row) => async () => {
        setOpenTransferDialog(true);
        context.actions.setVolume(row);
    };

    const handleEmpty = (row) => async () => {
        await emptyVolume(row.volumeName);
        await calculateVolumeSize(row.volumeName);
    };

    const handleDelete = (row) => async () => {
        setOpenDeleteForeverDialog(true);
        context.actions.setVolume(row);
    };

    const handleCellClick = (params: GridCellParams) => {
        if (params.colDef.field === "volumeName") {
            ddClient.desktopUI.navigate.viewVolume(params.row.volumeName);
        }
    };

    const {data: rows, isLoading, listVolumes, setData} = useGetVolumes();

    useEffect(() => {
        const volumeEvents = async () => {
            console.log("listening to volume events...");
            await ddClient.docker.cli.exec(
                "events",
                ["--format", `"{{ json . }}"`, "--filter", "type=volume"],
                {
                    stream: {
                        onOutput(data) {
                            listVolumes();
                        },
                        onClose(exitCode) {
                            console.log("onClose with exit code " + exitCode);
                        },
                        splitOutputLines: true,
                    },
                }
            );
        };

        volumeEvents();
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

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
            if (isError(output.stderr)) {
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

    const handleRunContainerDialogClose = () => {
        setOpenRunContainerDialog(false);
        context.actions.setVolume(null);
    };

    const handleExportDialogClose = () => {
        setOpenExportDialog(false);
        listVolumes();
        context.actions.setVolume(null);
    };

    const handleImportIntoNewDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenImportIntoNewDialog(false);
        context.actions.setVolume(null);
        if (actionSuccessfullyCompleted) {
            if (context.store.volume) calculateVolumeSize(context.store.volume.volumeName);
        }
    };


    const handleSaveDialogClose = () => {
        setOpenSaveDialog(false);
        context.actions.setVolume(null);
    };

    const handleLoadDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenLoadDialog(false);
        context.actions.setVolume(null);
        if (actionSuccessfullyCompleted) {
            calculateVolumeSize(context.store.volume.volumeName);
        }
    };

    const handleCloneDialogClose = (actionSuccessfullyCompleted: boolean) => {
        setOpenCloneDialog(false);
        context.actions.setVolume(null);
        if (actionSuccessfullyCompleted) {
            listVolumes();
        }
    };

    const handleTransferDialogClose = () => {
        setOpenTransferDialog(false);
        context.actions.setVolume(null);
    };

    const handleDeleteForeverDialogClose = (
        actionSuccessfullyCompleted: boolean
    ) => {
        setOpenDeleteForeverDialog(false);
        context.actions.setVolume(null);
        if (actionSuccessfullyCompleted) {
            listVolumes();
        }
    };

    const calculateVolumeSize = async (volumeName: string) => {
        let volumesSizeLoadingMapCopy = volumesSizeLoadingMap;
        volumesSizeLoadingMapCopy[volumeName] = true;
        setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy);

        try {
            ddClient.extension.vm.service
                .get(`/volumes/${volumeName}/size`)
                .then((res: any) => {
                    // e.g. {"Bytes":16000,"Human":"16.0 kB"}
                    const resJSON = JSON.stringify(res)
                    const sizeObj = JSON.parse(resJSON)
                    let rowsCopy = rows.slice(); // copy the array
                    const index = rowsCopy.findIndex(
                        (element) => element.volumeName === volumeName
                    );
                    rowsCopy[index].volumeSize = sizeObj.Human;

                    setData(rowsCopy);

                    let volumesSizeLoadingMapCopy = volumesSizeLoadingMap;
                    volumesSizeLoadingMapCopy[volumeName] = false;
                    setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy);
                });
        } catch (error) {
            ddClient.desktopUI.toast.error(
                `Failed to recalculate volume size: ${error.stderr}`
            );
        }
    };

    return (
        <>
            <Header />
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
                            loading={isLoading}
                            components={{
                                LoadingOverlay: LinearProgress,
                                Toolbar: () => <CustomToolbar openDialog={() => setOpenImportIntoNewDialog(true)} />,
                            }}
                            rows={rows || []}
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

                    {openImportIntoNewDialog && (
                        <ImportDialog
                            volumes={rows}
                            open={openImportIntoNewDialog}
                            onClose={handleImportIntoNewDialogClose}
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
                                handleDeleteForeverDialogClose(e);
                            }}
                        />
                    )}
                </Grid>
            </Stack>
        </>
    );
}
