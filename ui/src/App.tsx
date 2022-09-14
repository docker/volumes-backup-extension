import React, { useContext, useEffect } from "react";
import {
  DataGrid,
  GridActionsCellItem,
  GridCellParams,
  GridToolbarColumnsButton,
  GridToolbarContainer,
  GridToolbarDensitySelector,
  GridToolbarFilterButton,
} from "@mui/x-data-grid";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import {
  Box,
  Button,
  CircularProgress,
  Grid,
  LinearProgress,
  Skeleton,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import {
  CopyAll as CopyAllIcon,
  Delete as DeleteIcon,
  DeleteForever as DeleteForeverIcon,
  DesktopWindows as DesktopWindowsIcon,
  Download as DownloadIcon,
  Upload as UploadIcon,
  Visibility as VisibilityIcon,
} from "@mui/icons-material";
import { useNotificationContext } from "./NotificationContext";
import ExportDialog from "./components/ExportDialog";
import CloneDialog from "./components/CloneDialog";
import TransferDialog from "./components/TransferDialog";
import DeleteForeverDialog from "./components/DeleteForeverDialog";
import { MyContext } from ".";
import { isError } from "./common/isError";
import ImportDialog from "./components/ImportDialog";
import { useGetVolumes } from "./hooks/useGetVolumes";
import { Header } from "./components/Header";
import { track } from "./common/track";

const ddClient = createDockerDesktopClient();

interface ICustomToolbar {
  openDialog(): void;
}

function CustomToolbar({ openDialog }: ICustomToolbar) {
  return (
    <GridToolbarContainer>
      <Grid container justifyContent="space-between">
        <Grid item>
          <GridToolbarColumnsButton />
          <GridToolbarFilterButton />
          <GridToolbarDensitySelector />
        </Grid>
        <Grid item>
          <Button
            variant="contained"
            onClick={() => {
              track({ action: "ImportNewVolumePopup" });
              openDialog();
            }}
            endIcon={<DownloadIcon />}
          >
            Import into new volume
          </Button>
        </Grid>
      </Grid>
    </GridToolbarContainer>
  );
}

export function App() {
  const context = useContext(MyContext);
  const { sendNotification } = useNotificationContext();
  const [volumesSizeLoadingMap, setVolumesSizeLoadingMap] = React.useState<
    Record<string, boolean>
  >({});

  const [openExportDialog, setOpenExportDialog] =
    React.useState<boolean>(false);
  const [openImportIntoNewDialog, setOpenImportIntoNewDialog] =
    React.useState<boolean>(false);
  const [openCloneDialog, setOpenCloneDialog] = React.useState<boolean>(false);
  const [openTransferDialog, setOpenTransferDialog] =
    React.useState<boolean>(false);
  const [openDeleteForeverDialog, setOpenDeleteForeverDialog] =
    React.useState<boolean>(false);

  const [actionsInProgress, setActionsInProgress] = React.useState({});

  const [recalculateVolumeSize, setRecalculateVolumeSize] =
    React.useState<string>(null);

  const columns = [
    { field: "volumeDriver", headerName: "Driver", hide: true },
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
        if (isVolumesSizeLoading) {
          return (
            <Box sx={{ width: "100%" }}>
              <Skeleton animation="wave" />
            </Box>
          );
        }

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
        if (
          isVolumesSizeLoading ||
          volumesSizeLoadingMap[params.row.volumeName]
        ) {
          return (
            <Box sx={{ width: "100%" }}>
              <Skeleton animation="wave" />
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
      getActions: (params) => {
        if (params.row.volumeName in actionsInProgress) {
          const action = actionsInProgress[params.row.volumeName];
          return [
            <GridActionsCellItem
              className="circular-progress"
              key={"loading_" + params.row.id}
              icon={
                <>
                  <CircularProgress size={20} />
                  <Typography ml={2}>
                    {action.charAt(0).toUpperCase() + action.slice(1)} in
                    progress...
                  </Typography>
                </>
              }
              label="Loading"
              showInMenu={false}
            />,
          ];
        }

        return [
          <GridActionsCellItem
            key={"action_view_volume_" + params.row.id}
            icon={
              <Tooltip title="View volume">
                <VisibilityIcon>View volume</VisibilityIcon>
              </Tooltip>
            }
            label="View volume"
            onClick={handleNavigate(params.row)}
            showInMenu
          />,
          <GridActionsCellItem
            showInMenu
            key={"action_clone_volume_" + params.row.id}
            icon={
              <Tooltip title="Clone volume">
                <CopyAllIcon>Clone volume</CopyAllIcon>
              </Tooltip>
            }
            label="Clone volume"
            onClick={handleClone(params.row)}
          />,
          <GridActionsCellItem
            key={"action_export_" + params.row.id}
            icon={
              <Tooltip title="Export volume">
                <UploadIcon>Export volume</UploadIcon>
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
                <DownloadIcon>Import</DownloadIcon>
              </Tooltip>
            }
            label="Import"
            onClick={handleImport(params.row)}
          />,
          <GridActionsCellItem
            key={"action_transfer_" + params.row.id}
            icon={
              <Tooltip title="Transfer to host">
                <DesktopWindowsIcon>Transfer to host</DesktopWindowsIcon>
              </Tooltip>
            }
            label="Transfer to host"
            onClick={handleTransfer(params.row)}
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
            disabled={params.row.volumeContainers?.length > 0} // do not allow to delete volumes in use
            showInMenu
          />,
        ];
      },
    },
  ];

  const handleNavigate = (row) => async () => {
    track({ action: "ViewVolumeDetails" });
    ddClient.desktopUI.navigate.viewVolume(row.volumeName);
  };

  const handleClone = (row) => () => {
    track({ action: "CloneVolumePopup" });
    setOpenCloneDialog(true);
    context.actions.setVolume(row);
  };

  const handleExport = (row) => () => {
    track({ action: "ExportVolumePopup" });
    setOpenExportDialog(true);
    context.actions.setVolume(row);
  };

  const handleImport = (row) => () => {
    track({ action: "ImportVolumePopup" });
    context.actions.setVolume(row);
    setOpenImportIntoNewDialog(true);
  };

  const handleTransfer = (row) => async () => {
    track({ action: "TransferVolumePopup" });
    setOpenTransferDialog(true);
    context.actions.setVolume(row);
  };

  const handleEmpty = (row) => async () => {
    track({ action: "EmptyVolume" });
    await emptyVolume(row.volumeName);
    await calculateVolumeSize(row.volumeName);
  };

  const handleDelete = (row) => async () => {
    track({ action: "DeleteVolumePopup" });
    setOpenDeleteForeverDialog(true);
    context.actions.setVolume(row);
  };

  const handleCellClick = (params: GridCellParams) => {
    if (params.colDef.field === "volumeName") {
      track({ action: "NavigateToVolumeDetails" });
      ddClient.desktopUI.navigate.viewVolume(params.row.volumeName);
    }
  };

  const {
    data: rows,
    listVolumes,
    isLoading,
    isVolumesSizeLoading,
    setData,
  } = useGetVolumes();

  const getActionsInProgress = async () => {
    ddClient.extension.vm.service
      .get("/progress")
      .then((result: unknown) => {
        setActionsInProgress(result);
      })
      .catch((error) => {
        console.error(error);
      });
  };

  useEffect(() => {
    getActionsInProgress();
  }, []);

  useEffect(() => {
    const extensionContainersEvents = async () => {
      console.log("listening to extension's container events...");
      await ddClient.docker.cli.exec(
        "events",
        [
          "--format",
          `"{{ json . }}"`,
          "--filter",
          "type=container",
          "--filter",
          "event=create",
          "--filter",
          "event=destroy",
          "--filter",
          "label=com.docker.compose.project=docker_volumes-backup-extension-desktop-extension",
          "--filter",
          "label=com.volumes-backup-extension.trigger-ui-refresh=true",
        ],
        {
          stream: {
            async onOutput() {
              await getActionsInProgress();
            },
            onClose(exitCode) {
              console.log("onClose with exit code " + exitCode);
            },
            splitOutputLines: true,
          },
        }
      );
    };

    extensionContainersEvents();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const emptyVolume = (volumeName: string) => {
    return ddClient.docker.cli
      .exec("run", [
        "--rm",
        "--label com.volumes-backup-extension.trigger-ui-refresh=true",
        "--label com.docker.compose.project=docker_volumes-backup-extension-desktop-extension",
        `-v=${volumeName}:/vackup-volume `,
        "busybox",
        "/bin/sh",
        "-c",
        '"rm -rf /vackup-volume/..?* /vackup-volume/.[!.]* /vackup-volume/*"', // hidden and not-hidden files and folders: .[!.]* matches all dot files except . and files whose name begins with .., and ..?* matches all dot-dot files except ..
      ])
      .then((output) => {
        if (isError(output.stderr)) {
          sendNotification.error(output.stderr);
          return;
        }
        sendNotification.info(
          `The content of volume ${volumeName} has been removed`
        );
      })
      .catch((error) => {
        sendNotification.error(
          `Failed to empty volume ${volumeName}: ${error.stderr} Exit code: ${error.code}`
        );
      });
  };

  const handleExportDialogClose = () => {
    setOpenExportDialog(false);
    context.actions.setVolume(null);
  };

  const handleImportIntoNewDialogClose = () => {
    setOpenImportIntoNewDialog(false);
    context.actions.setVolume(null);
  };

  const handleImportIntoNewDialogCompletion = (
    actionSuccessfullyCompleted: boolean,
    selectedVolumeName: string
  ) => {
    if (actionSuccessfullyCompleted) {
      if (selectedVolumeName && context.store.volume) {
        // the import is performed on an existing volume
        calculateVolumeSize(context.store.volume.volumeName);
      } else {
        // the import is performed on a new volume, so we fetch all volumes to populate the table
        listVolumes();
      }
    }
  };

  const handleCloneDialogClose = () => {
    setOpenCloneDialog(false);
    context.actions.setVolume(null);
  };

  const handleCloneDialogOnCompletion = (
    clonedVolumeName: string,
    actionSuccessfullyCompleted: boolean
  ) => {
    if (actionSuccessfullyCompleted) {
      const rowsCopy = rows.slice();
      rowsCopy.push({
        id: rows.length,
        volumeName: clonedVolumeName,
        volumeDriver: "local",
      });

      setData(rowsCopy);
      setRecalculateVolumeSize(clonedVolumeName);
    }
  };

  useEffect(() => {
    if (!recalculateVolumeSize) {
      return;
    }
    calculateVolumeSize(recalculateVolumeSize);
  }, [recalculateVolumeSize]);

  const handleTransferDialogClose = () => {
    setOpenTransferDialog(false);
    context.actions.setVolume(null);
  };

  const handleDeleteForeverDialogClose = () => {
    setOpenDeleteForeverDialog(false);
    context.actions.setVolume(null);
  };

  const handleDeleteForeverDialogCompletion = (
    actionSuccessfullyCompleted: boolean
  ) => {
    if (actionSuccessfullyCompleted && context.store.volume) {
      const rowsCopy = rows.slice();
      const index = rowsCopy.findIndex(
        (element) => element.volumeName === context.store.volume.volumeName
      );
      if (index > -1) {
        rowsCopy.splice(index, 1);
      }

      setData(rowsCopy);
    }
  };

  const calculateVolumeSize = async (volumeName: string) => {
    const volumesSizeLoadingMapCopy = volumesSizeLoadingMap;
    volumesSizeLoadingMapCopy[volumeName] = true;
    setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy);

    try {
      ddClient.extension.vm.service
        .get(`/volumes/${volumeName}/size`)
        .then((res: unknown) => {
          // e.g. {"Bytes":16000,"Human":"16.0 kB"}
          const resJSON = JSON.stringify(res);
          const sizeObj = JSON.parse(resJSON);
          const rowsCopy = rows.slice(); // copy the array
          const index = rowsCopy.findIndex(
            (element) => element.volumeName === volumeName
          );
          rowsCopy[index].volumeSize = sizeObj.Human;

          setData(rowsCopy);

          const volumesSizeLoadingMapCopy = volumesSizeLoadingMap;
          volumesSizeLoadingMapCopy[volumeName] = false;
          setVolumesSizeLoadingMap(volumesSizeLoadingMapCopy);
        });
    } catch (error) {
      sendNotification.error(
        `Failed to recalculate volume size: ${error.stderr}`
      );
    }
  };

  return (
    <>
      <Header />
      <Stack direction="column" alignItems="start" spacing={2} sx={{ mt: 4 }}>
        <Grid container>
          <Grid item flex={1}>
            <DataGrid
              loading={isLoading}
              components={{
                LoadingOverlay: LinearProgress,
                Toolbar: () => (
                  <CustomToolbar
                    openDialog={() => setOpenImportIntoNewDialog(true)}
                  />
                ),
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
                "& .MuiDataGrid-cell": {
                  "& .MuiIconButton-root.circular-progress": {
                    "&:hover": {
                      backgroundColor: "transparent",
                    },
                    backgroundColor: "transparent",
                  },
                },
              }}
            />
          </Grid>

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
              onCompletion={handleImportIntoNewDialogCompletion}
            />
          )}

          {openCloneDialog && (
            <CloneDialog
              open={openCloneDialog}
              onClose={handleCloneDialogClose}
              onCompletion={handleCloneDialogOnCompletion}
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
              onClose={handleDeleteForeverDialogClose}
              onCompletion={handleDeleteForeverDialogCompletion}
            />
          )}
        </Grid>
      </Stack>
    </>
  );
}
