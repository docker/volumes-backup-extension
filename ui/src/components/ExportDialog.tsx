import React, { useEffect } from "react";
import {
  Button,
  TextField,
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

export default function ExportDialog({ ...props }) {
  console.log("ExportDialog component rendered.");
  const [fileName, setFileName] = React.useState<string>(props.volumeName);
  const [path, setPath] = React.useState<string>("");
  const [actionInProgress, setActionInProgress] =
    React.useState<boolean>(false);

  const ddClient = useDockerDesktopClient();

  useEffect(() => {
    setFileName(`${props.volumeName}.tar.gz`);
  }, [props.volumeName]);

  const selectExportDirectory = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openDirectory"],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const exportVolume = async () => {
    setActionInProgress(true);

    try {
      const output = await ddClient.docker.cli.exec("run", [
        "--rm",
        `-v=${props.volumeName}:/vackup-volume `,
        `-v=${path}:/vackup `,
        "busybox",
        "tar",
        "-zcvf",
        `/vackup/${fileName}`,
        "/vackup-volume",
      ]);
      if (output.stderr !== "") {
        //"tar: removing leading '/' from member names\n"
        if (!output.stderr.includes("tar: removing leading")) {
          // this is an error we may want to display
          ddClient.desktopUI.toast.error(output.stderr);
          return;
        }
      }
      ddClient.desktopUI.toast.success(
        `Volume ${props.volumeName} exported to ${path}`
      );
    } catch (error) {
      ddClient.desktopUI.toast.error(
        `Failed to backup volume ${props.volumeName} to ${path}: ${error.code}`
      );
    } finally {
      setActionInProgress(false);
      props.onClose();
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose}>
      <DialogTitle>Export volume to local directory</DialogTitle>
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
          Creates a gzip'ed tarball in the selected directory from a volume.
        </DialogContentText>

        <Grid container direction="column" spacing={2}>
          <Grid item>
            <TextField
              required
              autoFocus
              margin="dense"
              id="file-name"
              label="File name"
              fullWidth
              variant="standard"
              defaultValue={`${props.volumeName}.tar.gz`}
              spellCheck={false}
              onChange={(e) => {
                setFileName(e.target.value);
              }}
            />
          </Grid>
          <Grid item>
            <Button variant="contained" onClick={selectExportDirectory}>
              Select directory
            </Button>
          </Grid>

          {path !== "" && (
            <Grid item>
              <Typography variant="body1" color="text.secondary">
                The volume will be exported to {path}/{fileName}
              </Typography>
            </Grid>
          )}
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose}>Cancel</Button>
        <Button
          onClick={exportVolume}
          disabled={path === "" || fileName === ""}
        >
          Export
        </Button>
      </DialogActions>
    </Dialog>
  );
}
