import React, { useEffect } from "react";
import {
  Button,
  Typography,
  Grid,
  Backdrop,
  CircularProgress,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogContentText from "@mui/material/DialogContentText";
import DialogTitle from "@mui/material/DialogTitle";

import { createDockerDesktopClient } from "@docker/extension-api-client";

const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

export default function ImportDialog({ ...props }) {
  console.log("ImportDialog component rendered.");
  const [fileName, setFileName] = React.useState<string>(props.volumeName);
  const [path, setPath] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const ddClient = useDockerDesktopClient();

  useEffect(() => {
    setFileName(`${props.volumeName}.tar.gz`);
  }, [props.volumeName]);

  const selectImportTarGzFile = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openFile"],
        filters: [{ name: ".tar.gz", extensions: [".tar.gz"] }],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const importVolume = async () => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${props.volumeName}:/vackup-volume `,
        `-v=${path}:/vackup `, // path: e.g. "$HOME/Downloads/my-vol.tar.gz"
        "busybox",
        "tar",
        "-xvzf",
        `/vackup`,
      ]);
      if (output.stderr !== "") {
        ddClient.desktopUI.toast.error(output.stderr);
        return;
      }
      ddClient.desktopUI.toast.success(
        `File ${fileName} imported into volume ${props.volumeName}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to import file ${fileName} into volume ${props.volumeName}: ${error.stderr} Exit code: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
      setPath("");
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Import gzip'ed tarball into a volume</DialogTitle>
      <DialogContent>
        <Backdrop
          sx={{
            backgroundColor: "rgba(245,244,244,0.4)",
            zIndex: (theme) => theme.zIndex.drawer + 1,
          }}
          open={actionInProgress}
        >
          <CircularProgress color="info" />
        </Backdrop>
        <DialogContentText>
          Extracts a gzip'ed tarball into a volume.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
              Choose a .tar.gz to import, e.g. file.tar.gz
            </Typography>
          </Grid>
          <Grid item>
            <Button variant="contained" onClick={selectImportTarGzFile}>
              Select file
            </Button>
          </Grid>

          {path !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
                The file {path} will be imported into volume {props.volumeName}.
              </Typography>
              <Typography variant="body1" color="text.secondary">
                ⚠️ This will replace all the existing data inside the volume.
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button
          onClick={() => {
            setPath("");
            props.onClose();
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={importVolume}
          disabled={path === "" || fileName === ""}
        >
          Import
        </Button>
      </DialogActions>
    </Dialog>
  );
}
